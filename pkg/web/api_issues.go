package web

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/store"
	"github.com/gorilla/mux"
)

// ---- Response types --------------------------------------------------------

type issueResponse struct {
	ID          int64     `json:"id"`
	RepoID      int64     `json:"repo_id"`
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	Status      string    `json:"status"`
	AuthorID    int64     `json:"author_id"`
	MilestoneID *int64    `json:"milestone_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Labels      []string  `json:"labels"`
	Assignees   []string  `json:"assignees"`
}

type commentResponse struct {
	ID        int64     `json:"id"`
	IssueID   int64     `json:"issue_id"`
	AuthorID  int64     `json:"author_id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type labelResponse struct {
	ID          int64     `json:"id"`
	RepoID      int64     `json:"repo_id"`
	Name        string    `json:"name"`
	Color       string    `json:"color"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type milestoneResponse struct {
	ID          int64      `json:"id"`
	RepoID      int64      `json:"repo_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	ClosedAt    *time.Time `json:"closed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ---- Helpers ---------------------------------------------------------------

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func enrichIssue(ctx context.Context, be *backend.Backend, issue proto.Issue) issueResponse {
	resp := issueResponse{
		ID:        issue.ID(),
		RepoID:    issue.RepoID(),
		Title:     issue.Title(),
		Body:      issue.Body(),
		Status:    issue.Status(),
		AuthorID:  issue.UserID(),
		CreatedAt: issue.CreatedAt(),
		UpdatedAt: issue.UpdatedAt(),
		Labels:    []string{},
		Assignees: []string{},
	}

	if labels, err := be.GetIssueLabels(ctx, issue.ID()); err == nil {
		for _, l := range labels {
			resp.Labels = append(resp.Labels, l.Name())
		}
	}

	if assignees, err := be.GetIssueAssignees(ctx, issue.ID()); err == nil {
		for _, u := range assignees {
			resp.Assignees = append(resp.Assignees, u.Username())
		}
	}

	if ms, err := be.GetIssueMilestone(ctx, issue.ID()); err == nil && ms != nil {
		id := ms.ID()
		resp.MilestoneID = &id
	}

	return resp
}

func toLabelResponse(l proto.Label) labelResponse {
	return labelResponse{
		ID:          l.ID(),
		RepoID:      l.RepoID(),
		Name:        l.Name(),
		Color:       l.Color(),
		Description: l.Description(),
		CreatedAt:   l.CreatedAt(),
	}
}

func toMilestoneResponse(m proto.Milestone) milestoneResponse {
	resp := milestoneResponse{
		ID:          m.ID(),
		RepoID:      m.RepoID(),
		Title:       m.Title(),
		Description: m.Description(),
		CreatedAt:   m.CreatedAt(),
		UpdatedAt:   m.UpdatedAt(),
	}
	if !m.DueDate().IsZero() {
		t := m.DueDate()
		resp.DueDate = &t
	}
	if !m.ClosedAt().IsZero() {
		t := m.ClosedAt()
		resp.ClosedAt = &t
	}
	return resp
}

func toCommentResponse(c proto.IssueComment) commentResponse {
	return commentResponse{
		ID:        c.ID(),
		IssueID:   c.IssueID(),
		AuthorID:  c.UserID(),
		Body:      c.Body(),
		CreatedAt: c.CreatedAt(),
		UpdatedAt: c.UpdatedAt(),
	}
}

// getRepoAndCheckAccess extracts {repo} from mux vars, looks up the repo,
// authenticates the user (nil = anonymous), and checks minAccess.
// Returns repo, user (may be nil), and ok=false when an error response was already sent.
func getRepoAndCheckAccess(w http.ResponseWriter, r *http.Request, minAccess access.AccessLevel) (proto.Repository, proto.User, bool) {
	ctx := r.Context()
	be := backend.FromContext(ctx)
	repoName := mux.Vars(r)["repo"]

	user, _ := authenticate(r)

	repo, err := be.Repository(ctx, repoName)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return nil, nil, false
	}

	level := be.AccessLevelForUser(ctx, repoName, user)
	if level < minAccess {
		if user == nil {
			writeError(w, http.StatusUnauthorized, "authentication required")
		} else {
			writeError(w, http.StatusForbidden, "forbidden")
		}
		return nil, nil, false
	}

	return repo, user, true
}

// parseIssueID parses the {id} mux var as int64.
func parseIssueID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	return id, err == nil
}

// ---- Controller registration -----------------------------------------------

// IssueAPIController registers all issue API routes onto the router.
func IssueAPIController(ctx context.Context, r *mux.Router) {
	api := r.PathPrefix("/api/v1/repos/{repo}").Subrouter()

	// Issues
	api.HandleFunc("/issues", handleListIssues).Methods("GET")
	api.HandleFunc("/issues", handleCreateIssue).Methods("POST")
	api.HandleFunc("/issues/{id:[0-9]+}", handleGetIssue).Methods("GET")
	api.HandleFunc("/issues/{id:[0-9]+}", handleUpdateIssue).Methods("PATCH")
	api.HandleFunc("/issues/{id:[0-9]+}", handleDeleteIssue).Methods("DELETE")
	api.HandleFunc("/issues/{id:[0-9]+}/close", handleCloseIssue).Methods("POST")
	api.HandleFunc("/issues/{id:[0-9]+}/reopen", handleReopenIssue).Methods("POST")

	// Comments
	api.HandleFunc("/issues/{id:[0-9]+}/comments", handleListComments).Methods("GET")
	api.HandleFunc("/issues/{id:[0-9]+}/comments", handleCreateComment).Methods("POST")
	api.HandleFunc("/issues/{id:[0-9]+}/comments/{commentId:[0-9]+}", handleUpdateComment).Methods("PATCH")
	api.HandleFunc("/issues/{id:[0-9]+}/comments/{commentId:[0-9]+}", handleDeleteComment).Methods("DELETE")

	// Labels on issues
	api.HandleFunc("/issues/{id:[0-9]+}/labels", handleListIssueLabels).Methods("GET")
	api.HandleFunc("/issues/{id:[0-9]+}/labels", handleAddIssueLabel).Methods("POST")
	api.HandleFunc("/issues/{id:[0-9]+}/labels/{labelName}", handleRemoveIssueLabel).Methods("DELETE")

	// Assignees
	api.HandleFunc("/issues/{id:[0-9]+}/assignees", handleListAssignees).Methods("GET")
	api.HandleFunc("/issues/{id:[0-9]+}/assignees", handleAddAssignee).Methods("POST")
	api.HandleFunc("/issues/{id:[0-9]+}/assignees/{username}", handleRemoveAssignee).Methods("DELETE")

	// Repo labels
	api.HandleFunc("/labels", handleListLabels).Methods("GET")
	api.HandleFunc("/labels", handleCreateLabel).Methods("POST")
	api.HandleFunc("/labels/{id:[0-9]+}", handleGetLabel).Methods("GET")
	api.HandleFunc("/labels/{id:[0-9]+}", handleUpdateLabel).Methods("PATCH")
	api.HandleFunc("/labels/{id:[0-9]+}", handleDeleteLabel).Methods("DELETE")

	// Milestones
	api.HandleFunc("/milestones", handleListMilestones).Methods("GET")
	api.HandleFunc("/milestones", handleCreateMilestone).Methods("POST")
	api.HandleFunc("/milestones/{id:[0-9]+}", handleGetMilestone).Methods("GET")
	api.HandleFunc("/milestones/{id:[0-9]+}", handleUpdateMilestone).Methods("PATCH")
	api.HandleFunc("/milestones/{id:[0-9]+}", handleDeleteMilestone).Methods("DELETE")
	api.HandleFunc("/milestones/{id:[0-9]+}/close", handleCloseMilestone).Methods("POST")
	api.HandleFunc("/milestones/{id:[0-9]+}/reopen", handleReopenMilestone).Methods("POST")
}

// ---- Issue handlers --------------------------------------------------------

func handleListIssues(w http.ResponseWriter, r *http.Request) {
	repo, _, ok := getRepoAndCheckAccess(w, r, access.ReadOnlyAccess)
	if !ok {
		return
	}

	ctx := r.Context()
	be := backend.FromContext(ctx)
	q := r.URL.Query()

	filter := store.IssueFilter{
		Status:    q.Get("status"),
		Search:    q.Get("search"),
		LabelName: q.Get("label"),
	}
	if filter.Status == "" {
		filter.Status = "open"
	}
	if ms := q.Get("milestone"); ms != "" {
		if id, err := strconv.ParseInt(ms, 10, 64); err == nil {
			filter.MilestoneID = id
		}
	}
	if p := q.Get("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil {
			filter.Page = n
		}
	}
	if l := q.Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil {
			filter.Limit = n
		}
	}

	issues, err := be.GetIssuesByRepository(ctx, repo.Name(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	resp := make([]issueResponse, len(issues))
	for i, issue := range issues {
		resp[i] = enrichIssue(ctx, be, issue)
	}
	writeJSON(w, http.StatusOK, resp)
}

func handleCreateIssue(w http.ResponseWriter, r *http.Request) {
	repo, user, ok := getRepoAndCheckAccess(w, r, access.ReadOnlyAccess)
	if !ok {
		return
	}
	if user == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	ctx := r.Context()
	be := backend.FromContext(ctx)

	var body struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}
	if body.Title == "" {
		writeError(w, http.StatusUnprocessableEntity, "title is required")
		return
	}

	issue, err := be.CreateIssue(ctx, repo.Name(), user.ID(), body.Title, body.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, enrichIssue(ctx, be, issue))
}

func handleGetIssue(w http.ResponseWriter, r *http.Request) {
	repo, _, ok := getRepoAndCheckAccess(w, r, access.ReadOnlyAccess)
	if !ok {
		return
	}

	id, ok2 := parseIssueID(r)
	if !ok2 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	ctx := r.Context()
	be := backend.FromContext(ctx)

	issue, err := be.GetIssue(ctx, id)
	if err != nil || issue.RepoID() != repo.ID() {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	writeJSON(w, http.StatusOK, enrichIssue(ctx, be, issue))
}

func handleUpdateIssue(w http.ResponseWriter, r *http.Request) {
	repo, user, ok := getRepoAndCheckAccess(w, r, access.ReadOnlyAccess)
	if !ok {
		return
	}
	if user == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	id, ok2 := parseIssueID(r)
	if !ok2 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	ctx := r.Context()
	be := backend.FromContext(ctx)

	issue, err := be.GetIssue(ctx, id)
	if err != nil || issue.RepoID() != repo.ID() {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	// Auth: issue author or admin
	level := be.AccessLevelForUser(ctx, repo.Name(), user)
	if issue.UserID() != user.ID() && level < access.AdminAccess {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	var body struct {
		Title *string `json:"title"`
		Body  *string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}

	title := issue.Title()
	if body.Title != nil {
		title = *body.Title
	}
	if title == "" {
		writeError(w, http.StatusUnprocessableEntity, "title cannot be empty")
		return
	}

	if err := be.UpdateIssue(ctx, id, repo.ID(), title, body.Body); err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	updated, err := be.GetIssue(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, enrichIssue(ctx, be, updated))
}

func handleDeleteIssue(w http.ResponseWriter, r *http.Request) {
	repo, user, ok := getRepoAndCheckAccess(w, r, access.ReadOnlyAccess)
	if !ok {
		return
	}
	if user == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	id, ok2 := parseIssueID(r)
	if !ok2 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	ctx := r.Context()
	be := backend.FromContext(ctx)

	issue, err := be.GetIssue(ctx, id)
	if err != nil || issue.RepoID() != repo.ID() {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	level := be.AccessLevelForUser(ctx, repo.Name(), user)
	if issue.UserID() != user.ID() && level < access.AdminAccess {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	if err := be.DeleteIssue(ctx, id, repo.ID()); err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleCloseIssue(w http.ResponseWriter, r *http.Request) {
	repo, user, ok := getRepoAndCheckAccess(w, r, access.ReadWriteAccess)
	if !ok {
		return
	}
	if user == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	id, ok2 := parseIssueID(r)
	if !ok2 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	ctx := r.Context()
	be := backend.FromContext(ctx)

	issue, err := be.GetIssue(ctx, id)
	if err != nil || issue.RepoID() != repo.ID() {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	if err := be.CloseIssue(ctx, id, repo.ID(), user.ID()); err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	updated, err := be.GetIssue(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, enrichIssue(ctx, be, updated))
}

func handleReopenIssue(w http.ResponseWriter, r *http.Request) {
	repo, user, ok := getRepoAndCheckAccess(w, r, access.ReadWriteAccess)
	if !ok {
		return
	}
	if user == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	id, ok2 := parseIssueID(r)
	if !ok2 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	ctx := r.Context()
	be := backend.FromContext(ctx)

	issue, err := be.GetIssue(ctx, id)
	if err != nil || issue.RepoID() != repo.ID() {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	if err := be.ReopenIssue(ctx, id, repo.ID()); err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	updated, err := be.GetIssue(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, enrichIssue(ctx, be, updated))
}

// ---- Comment handlers ------------------------------------------------------

func handleListComments(w http.ResponseWriter, r *http.Request) {
	repo, _, ok := getRepoAndCheckAccess(w, r, access.ReadOnlyAccess)
	if !ok {
		return
	}

	id, ok2 := parseIssueID(r)
	if !ok2 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	ctx := r.Context()
	be := backend.FromContext(ctx)

	issue, err := be.GetIssue(ctx, id)
	if err != nil || issue.RepoID() != repo.ID() {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	comments, err := be.GetIssueComments(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	resp := make([]commentResponse, len(comments))
	for i, c := range comments {
		resp[i] = toCommentResponse(c)
	}
	writeJSON(w, http.StatusOK, resp)
}

func handleCreateComment(w http.ResponseWriter, r *http.Request) {
	repo, user, ok := getRepoAndCheckAccess(w, r, access.ReadOnlyAccess)
	if !ok {
		return
	}
	if user == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	id, ok2 := parseIssueID(r)
	if !ok2 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	ctx := r.Context()
	be := backend.FromContext(ctx)

	issue, err := be.GetIssue(ctx, id)
	if err != nil || issue.RepoID() != repo.ID() {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	var body struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}
	if body.Body == "" {
		writeError(w, http.StatusUnprocessableEntity, "body is required")
		return
	}

	comment, err := be.AddIssueComment(ctx, id, user.ID(), body.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, toCommentResponse(comment))
}

func handleUpdateComment(w http.ResponseWriter, r *http.Request) {
	repo, user, ok := getRepoAndCheckAccess(w, r, access.ReadOnlyAccess)
	if !ok {
		return
	}
	if user == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	issueID, ok2 := parseIssueID(r)
	if !ok2 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	commentIDStr := mux.Vars(r)["commentId"]
	commentID, err := strconv.ParseInt(commentIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	ctx := r.Context()
	be := backend.FromContext(ctx)

	issue, err := be.GetIssue(ctx, issueID)
	if err != nil || issue.RepoID() != repo.ID() {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	comment, err := be.GetIssueComment(ctx, commentID)
	if err != nil || comment.IssueID() != issueID {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	level := be.AccessLevelForUser(ctx, repo.Name(), user)
	if comment.UserID() != user.ID() && level < access.AdminAccess {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	var body struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}
	if body.Body == "" {
		writeError(w, http.StatusUnprocessableEntity, "body is required")
		return
	}

	if err := be.UpdateIssueComment(ctx, commentID, body.Body); err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	updated, err := be.GetIssueComment(ctx, commentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, toCommentResponse(updated))
}

func handleDeleteComment(w http.ResponseWriter, r *http.Request) {
	repo, user, ok := getRepoAndCheckAccess(w, r, access.ReadOnlyAccess)
	if !ok {
		return
	}
	if user == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	issueID, ok2 := parseIssueID(r)
	if !ok2 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	commentIDStr := mux.Vars(r)["commentId"]
	commentID, err := strconv.ParseInt(commentIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	ctx := r.Context()
	be := backend.FromContext(ctx)

	issue, err := be.GetIssue(ctx, issueID)
	if err != nil || issue.RepoID() != repo.ID() {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	comment, err := be.GetIssueComment(ctx, commentID)
	if err != nil || comment.IssueID() != issueID {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	level := be.AccessLevelForUser(ctx, repo.Name(), user)
	if comment.UserID() != user.ID() && level < access.AdminAccess {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	if err := be.DeleteIssueComment(ctx, commentID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- Issue label handlers --------------------------------------------------

func handleListIssueLabels(w http.ResponseWriter, r *http.Request) {
	repo, _, ok := getRepoAndCheckAccess(w, r, access.ReadOnlyAccess)
	if !ok {
		return
	}

	id, ok2 := parseIssueID(r)
	if !ok2 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	ctx := r.Context()
	be := backend.FromContext(ctx)

	issue, err := be.GetIssue(ctx, id)
	if err != nil || issue.RepoID() != repo.ID() {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	labels, err := be.GetIssueLabels(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	resp := make([]labelResponse, len(labels))
	for i, l := range labels {
		resp[i] = toLabelResponse(l)
	}
	writeJSON(w, http.StatusOK, resp)
}

func handleAddIssueLabel(w http.ResponseWriter, r *http.Request) {
	repo, user, ok := getRepoAndCheckAccess(w, r, access.ReadWriteAccess)
	if !ok {
		return
	}
	if user == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	id, ok2 := parseIssueID(r)
	if !ok2 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	ctx := r.Context()
	be := backend.FromContext(ctx)

	issue, err := be.GetIssue(ctx, id)
	if err != nil || issue.RepoID() != repo.ID() {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}
	if body.Name == "" {
		writeError(w, http.StatusUnprocessableEntity, "name is required")
		return
	}

	label, err := be.GetLabel(ctx, repo.Name(), body.Name)
	if err != nil {
		writeError(w, http.StatusNotFound, "label not found")
		return
	}

	if err := be.AddLabelToIssue(ctx, id, label.ID()); err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.WriteHeader(http.StatusOK)
}

func handleRemoveIssueLabel(w http.ResponseWriter, r *http.Request) {
	repo, user, ok := getRepoAndCheckAccess(w, r, access.ReadWriteAccess)
	if !ok {
		return
	}
	if user == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	id, ok2 := parseIssueID(r)
	if !ok2 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	ctx := r.Context()
	be := backend.FromContext(ctx)
	labelName := mux.Vars(r)["labelName"]

	issue, err := be.GetIssue(ctx, id)
	if err != nil || issue.RepoID() != repo.ID() {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	label, err := be.GetLabel(ctx, repo.Name(), labelName)
	if err != nil {
		writeError(w, http.StatusNotFound, "label not found")
		return
	}

	if err := be.RemoveLabelFromIssue(ctx, id, label.ID()); err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- Assignee handlers -----------------------------------------------------

func handleListAssignees(w http.ResponseWriter, r *http.Request) {
	repo, _, ok := getRepoAndCheckAccess(w, r, access.ReadOnlyAccess)
	if !ok {
		return
	}

	id, ok2 := parseIssueID(r)
	if !ok2 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	ctx := r.Context()
	be := backend.FromContext(ctx)

	issue, err := be.GetIssue(ctx, id)
	if err != nil || issue.RepoID() != repo.ID() {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	assignees, err := be.GetIssueAssignees(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	names := make([]string, len(assignees))
	for i, u := range assignees {
		names[i] = u.Username()
	}
	writeJSON(w, http.StatusOK, names)
}

func handleAddAssignee(w http.ResponseWriter, r *http.Request) {
	repo, user, ok := getRepoAndCheckAccess(w, r, access.ReadWriteAccess)
	if !ok {
		return
	}
	if user == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	id, ok2 := parseIssueID(r)
	if !ok2 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	ctx := r.Context()
	be := backend.FromContext(ctx)

	issue, err := be.GetIssue(ctx, id)
	if err != nil || issue.RepoID() != repo.ID() {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	var body struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}
	if body.Username == "" {
		writeError(w, http.StatusUnprocessableEntity, "username is required")
		return
	}

	if err := be.AssignUserToIssue(ctx, repo.Name(), id, body.Username); err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.WriteHeader(http.StatusOK)
}

func handleRemoveAssignee(w http.ResponseWriter, r *http.Request) {
	repo, user, ok := getRepoAndCheckAccess(w, r, access.ReadWriteAccess)
	if !ok {
		return
	}
	if user == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	id, ok2 := parseIssueID(r)
	if !ok2 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	ctx := r.Context()
	be := backend.FromContext(ctx)
	username := mux.Vars(r)["username"]

	issue, err := be.GetIssue(ctx, id)
	if err != nil || issue.RepoID() != repo.ID() {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	if err := be.UnassignUserFromIssue(ctx, repo.Name(), id, username); err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- Repo label handlers ---------------------------------------------------

func handleListLabels(w http.ResponseWriter, r *http.Request) {
	repo, _, ok := getRepoAndCheckAccess(w, r, access.ReadOnlyAccess)
	if !ok {
		return
	}

	ctx := r.Context()
	be := backend.FromContext(ctx)

	labels, err := be.ListLabels(ctx, repo.Name())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	resp := make([]labelResponse, len(labels))
	for i, l := range labels {
		resp[i] = toLabelResponse(l)
	}
	writeJSON(w, http.StatusOK, resp)
}

func handleCreateLabel(w http.ResponseWriter, r *http.Request) {
	repo, user, ok := getRepoAndCheckAccess(w, r, access.AdminAccess)
	if !ok {
		return
	}
	if user == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	ctx := r.Context()
	be := backend.FromContext(ctx)

	var body struct {
		Name        string `json:"name"`
		Color       string `json:"color"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}
	if body.Name == "" {
		writeError(w, http.StatusUnprocessableEntity, "name is required")
		return
	}

	label, err := be.CreateLabel(ctx, repo.Name(), body.Name, body.Color, body.Description)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, toLabelResponse(label))
}

func handleGetLabel(w http.ResponseWriter, r *http.Request) {
	repo, _, ok := getRepoAndCheckAccess(w, r, access.ReadOnlyAccess)
	if !ok {
		return
	}

	labelID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	ctx := r.Context()
	datastore := store.FromContext(ctx)
	dbx := db.FromContext(ctx)

	m, err := datastore.GetLabelByID(ctx, dbx, labelID)
	if err != nil || m.RepoID != repo.ID() {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	writeJSON(w, http.StatusOK, labelResponse{
		ID:          m.ID,
		RepoID:      m.RepoID,
		Name:        m.Name,
		Color:       m.Color,
		Description: m.Description,
		CreatedAt:   m.CreatedAt,
	})
}

func handleUpdateLabel(w http.ResponseWriter, r *http.Request) {
	repo, user, ok := getRepoAndCheckAccess(w, r, access.AdminAccess)
	if !ok {
		return
	}
	if user == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	labelID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	ctx := r.Context()
	be := backend.FromContext(ctx)
	datastore := store.FromContext(ctx)
	dbx := db.FromContext(ctx)

	existing, err := datastore.GetLabelByID(ctx, dbx, labelID)
	if err != nil || existing.RepoID != repo.ID() {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	var body struct {
		Name        *string `json:"name"`
		Color       *string `json:"color"`
		Description *string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}

	name := existing.Name
	if body.Name != nil {
		name = *body.Name
	}
	color := existing.Color
	if body.Color != nil {
		color = *body.Color
	}
	description := existing.Description
	if body.Description != nil {
		description = *body.Description
	}

	if err := be.UpdateLabel(ctx, repo.Name(), labelID, name, color, description); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	updated, err := datastore.GetLabelByID(ctx, dbx, labelID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, labelResponse{
		ID:          updated.ID,
		RepoID:      updated.RepoID,
		Name:        updated.Name,
		Color:       updated.Color,
		Description: updated.Description,
		CreatedAt:   updated.CreatedAt,
	})
}

func handleDeleteLabel(w http.ResponseWriter, r *http.Request) {
	repo, user, ok := getRepoAndCheckAccess(w, r, access.AdminAccess)
	if !ok {
		return
	}
	if user == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	labelID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	ctx := r.Context()
	be := backend.FromContext(ctx)

	if err := be.DeleteLabel(ctx, repo.Name(), labelID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- Milestone handlers ----------------------------------------------------

func handleListMilestones(w http.ResponseWriter, r *http.Request) {
	repo, _, ok := getRepoAndCheckAccess(w, r, access.ReadOnlyAccess)
	if !ok {
		return
	}

	ctx := r.Context()
	be := backend.FromContext(ctx)

	// closed=true shows closed milestones; default shows open
	showClosed := r.URL.Query().Get("closed") == "true"

	milestones, err := be.ListMilestones(ctx, repo.Name(), !showClosed)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	resp := make([]milestoneResponse, len(milestones))
	for i, m := range milestones {
		resp[i] = toMilestoneResponse(m)
	}
	writeJSON(w, http.StatusOK, resp)
}

func handleCreateMilestone(w http.ResponseWriter, r *http.Request) {
	repo, user, ok := getRepoAndCheckAccess(w, r, access.AdminAccess)
	if !ok {
		return
	}
	if user == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	ctx := r.Context()
	be := backend.FromContext(ctx)

	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}
	if body.Title == "" {
		writeError(w, http.StatusUnprocessableEntity, "title is required")
		return
	}

	ms, err := be.CreateMilestone(ctx, repo.Name(), body.Title, body.Description)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, toMilestoneResponse(ms))
}

func handleGetMilestone(w http.ResponseWriter, r *http.Request) {
	repo, _, ok := getRepoAndCheckAccess(w, r, access.ReadOnlyAccess)
	if !ok {
		return
	}

	msID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	ctx := r.Context()
	be := backend.FromContext(ctx)

	ms, err := be.GetMilestone(ctx, repo.Name(), msID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	writeJSON(w, http.StatusOK, toMilestoneResponse(ms))
}

func handleUpdateMilestone(w http.ResponseWriter, r *http.Request) {
	repo, user, ok := getRepoAndCheckAccess(w, r, access.AdminAccess)
	if !ok {
		return
	}
	if user == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	msID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	ctx := r.Context()
	be := backend.FromContext(ctx)

	existing, err := be.GetMilestone(ctx, repo.Name(), msID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	var body struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}

	title := existing.Title()
	if body.Title != nil {
		title = *body.Title
	}
	description := existing.Description()
	if body.Description != nil {
		description = *body.Description
	}

	if err := be.UpdateMilestone(ctx, repo.Name(), msID, title, description); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	updated, err := be.GetMilestone(ctx, repo.Name(), msID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, toMilestoneResponse(updated))
}

func handleDeleteMilestone(w http.ResponseWriter, r *http.Request) {
	repo, user, ok := getRepoAndCheckAccess(w, r, access.AdminAccess)
	if !ok {
		return
	}
	if user == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	msID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	ctx := r.Context()
	be := backend.FromContext(ctx)

	if err := be.DeleteMilestone(ctx, repo.Name(), msID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleCloseMilestone(w http.ResponseWriter, r *http.Request) {
	repo, user, ok := getRepoAndCheckAccess(w, r, access.AdminAccess)
	if !ok {
		return
	}
	if user == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	msID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	ctx := r.Context()
	be := backend.FromContext(ctx)

	if _, err := be.GetMilestone(ctx, repo.Name(), msID); err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	if err := be.CloseMilestone(ctx, repo.Name(), msID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	updated, err := be.GetMilestone(ctx, repo.Name(), msID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, toMilestoneResponse(updated))
}

func handleReopenMilestone(w http.ResponseWriter, r *http.Request) {
	repo, user, ok := getRepoAndCheckAccess(w, r, access.AdminAccess)
	if !ok {
		return
	}
	if user == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	msID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	ctx := r.Context()
	be := backend.FromContext(ctx)

	if _, err := be.GetMilestone(ctx, repo.Name(), msID); err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	if err := be.ReopenMilestone(ctx, repo.Name(), msID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	updated, err := be.GetMilestone(ctx, repo.Name(), msID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, toMilestoneResponse(updated))
}
