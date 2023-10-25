package webhook

import "time"

// EventPayload is a webhook event payload.
type EventPayload interface {
	// Event returns the event type.
	Event() Event
	// RepositoryID returns the repository ID.
	RepositoryID() int64
}

// Common is a common payload.
type Common struct {
	// EventType is the event type.
	EventType Event `json:"event" url:"event"`
	// Repository is the repository payload.
	Repository Repository `json:"repository" url:"repository"`
	// Sender is the sender payload.
	Sender User `json:"sender" url:"sender"`
}

// Event returns the event type.
// Implements EventPayload.
func (c Common) Event() Event {
	return c.EventType
}

// RepositoryID returns the repository ID.
// Implements EventPayload.
func (c Common) RepositoryID() int64 {
	return c.Repository.ID
}

// User represents a user in an event.
type User struct {
	// ID is the owner ID.
	ID int64 `json:"id" url:"id"`
	// Username is the owner username.
	Username string `json:"username" url:"username"`
}

// Repository represents an event repository.
type Repository struct {
	// ID is the repository ID.
	ID int64 `json:"id" url:"id"`
	// Name is the repository name.
	Name string `json:"name" url:"name"`
	// ProjectName is the repository project name.
	ProjectName string `json:"project_name" url:"project_name"`
	// Description is the repository description.
	Description string `json:"description" url:"description"`
	// DefaultBranch is the repository default branch.
	DefaultBranch string `json:"default_branch" url:"default_branch"`
	// Private is whether the repository is private.
	Private bool `json:"private" url:"private"`
	// Owner is the repository owner.
	Owner User `json:"owner" url:"owner"`
	// HTTPURL is the repository HTTP URL.
	HTTPURL string `json:"http_url" url:"http_url"`
	// SSHURL is the repository SSH URL.
	SSHURL string `json:"ssh_url" url:"ssh_url"`
	// GitURL is the repository Git URL.
	GitURL string `json:"git_url" url:"git_url"`
	// CreatedAt is the repository creation time.
	CreatedAt time.Time `json:"created_at" url:"created_at"`
	// UpdatedAt is the repository last update time.
	UpdatedAt time.Time `json:"updated_at" url:"updated_at"`
}

// Author is a commit author.
type Author struct {
	// Name is the author name.
	Name string `json:"name" url:"name"`
	// Email is the author email.
	Email string `json:"email" url:"email"`
	// Date is the author date.
	Date time.Time `json:"date" url:"date"`
}

// Commit represents a Git commit.
type Commit struct {
	// ID is the commit ID.
	ID string `json:"id" url:"id"`
	// Message is the commit message.
	Message string `json:"message" url:"message"`
	// Title is the commit title.
	Title string `json:"title" url:"title"`
	// Author is the commit author.
	Author Author `json:"author" url:"author"`
	// Committer is the commit committer.
	Committer Author `json:"committer" url:"committer"`
	// Timestamp is the commit timestamp.
	Timestamp time.Time `json:"timestamp" url:"timestamp"`
}
