package lfs

import (
	"time"
)

const (
	// MediaType contains the media type for LFS server requests.
	MediaType = "application/vnd.git-lfs+json"

	// OperationDownload is the operation name for a download request.
	OperationDownload = "download"

	// OperationUpload is the operation name for an upload request.
	OperationUpload = "upload"

	// ActionDownload is the action name for a download request.
	ActionDownload = OperationDownload

	// ActionUpload is the action name for an upload request.
	ActionUpload = OperationUpload

	// ActionVerify is the action name for a verify request.
	ActionVerify = "verify"

	// DefaultLocksLimit is the default number of locks to return in a single
	// request.
	DefaultLocksLimit = 20
)

// Pointer contains LFS pointer data
type Pointer struct {
	Oid  string `json:"oid"`
	Size int64  `json:"size"`
}

// PointerBlob associates a Git blob with a Pointer.
type PointerBlob struct {
	Hash string
	Pointer
}

// ErrorResponse describes the error to the client.
type ErrorResponse struct {
	Message          string `json:"message,omitempty"`
	DocumentationURL string `json:"documentation_url,omitempty"`
	RequestID        string `json:"request_id,omitempty"`
}

// BatchResponse contains multiple object metadata Representation structures
// for use with the batch API.
// https://github.com/git-lfs/git-lfs/blob/main/docs/api/batch.md#successful-responses
type BatchResponse struct {
	Transfer string            `json:"transfer,omitempty"`
	Objects  []*ObjectResponse `json:"objects"`
	HashAlgo string            `json:"hash_algo,omitempty"`
}

// ObjectResponse is object metadata as seen by clients of the LFS server.
type ObjectResponse struct {
	Pointer
	Actions map[string]*Link `json:"actions,omitempty"`
	Error   *ObjectError     `json:"error,omitempty"`
}

// Link provides a structure with information about how to access a object.
type Link struct {
	Href      string            `json:"href"`
	Header    map[string]string `json:"header,omitempty"`
	ExpiresAt *time.Time        `json:"expires_at,omitempty"`
	ExpiresIn *time.Duration    `json:"expires_in,omitempty"`
}

// ObjectError defines the JSON structure returned to the client in case of an error.
type ObjectError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// BatchRequest contains multiple requests processed in one batch operation.
// https://github.com/git-lfs/git-lfs/blob/main/docs/api/batch.md#requests
type BatchRequest struct {
	Operation string     `json:"operation"`
	Transfers []string   `json:"transfers,omitempty"`
	Ref       *Reference `json:"ref,omitempty"`
	Objects   []Pointer  `json:"objects"`
	HashAlgo  string     `json:"hash_algo,omitempty"`
}

// Reference contains a git reference.
// https://github.com/git-lfs/git-lfs/blob/main/docs/api/batch.md#ref-property
type Reference struct {
	Name string `json:"name"`
}

// AuthenticateResponse is the git-lfs-authenticate JSON response object.
type AuthenticateResponse struct {
	Header    map[string]string `json:"header"`
	Href      string            `json:"href"`
	ExpiresIn time.Duration     `json:"expires_in"`
	ExpiresAt time.Time         `json:"expires_at"`
}

// LockCreateRequest contains the request data for creating a lock.
// https://github.com/git-lfs/git-lfs/blob/main/docs/api/locking.md
// https://github.com/git-lfs/git-lfs/blob/main/locking/schemas/http-lock-create-request-schema.json
type LockCreateRequest struct {
	Path string    `json:"path"`
	Ref  Reference `json:"ref,omitempty"`
}

// Owner contains the owner data for a lock.
type Owner struct {
	Name string `json:"name"`
}

// Lock contains the response data for creating a lock.
// https://github.com/git-lfs/git-lfs/blob/main/docs/api/locking.md
// https://github.com/git-lfs/git-lfs/blob/main/locking/schemas/http-lock-create-response-schema.json
type Lock struct {
	ID       string    `json:"id"`
	Path     string    `json:"path"`
	LockedAt time.Time `json:"locked_at"`
	Owner    Owner     `json:"owner,omitempty"`
}

// LockDeleteRequest contains the request data for deleting a lock.
// https://github.com/git-lfs/git-lfs/blob/main/docs/api/locking.md
// https://github.com/git-lfs/git-lfs/blob/main/locking/schemas/http-lock-delete-request-schema.json
type LockDeleteRequest struct {
	Force bool      `json:"force,omitempty"`
	Ref   Reference `json:"ref,omitempty"`
}

// LockListResponse contains the response data for listing locks.
// https://github.com/git-lfs/git-lfs/blob/main/docs/api/locking.md
// https://github.com/git-lfs/git-lfs/blob/main/locking/schemas/http-lock-list-response-schema.json
type LockListResponse struct {
	Locks      []Lock `json:"locks"`
	NextCursor string `json:"next_cursor,omitempty"`
}

// LockVerifyRequest contains the request data for verifying a lock.
type LockVerifyRequest struct {
	Ref    Reference `json:"ref,omitempty"`
	Cursor string    `json:"cursor,omitempty"`
	Limit  int       `json:"limit,omitempty"`
}

// LockVerifyResponse contains the response data for verifying a lock.
// https://github.com/git-lfs/git-lfs/blob/main/docs/api/locking.md
// https://github.com/git-lfs/git-lfs/blob/main/locking/schemas/http-lock-verify-response-schema.json
type LockVerifyResponse struct {
	Ours       []Lock `json:"ours"`
	Theirs     []Lock `json:"theirs"`
	NextCursor string `json:"next_cursor,omitempty"`
}

// LockResponse contains the response data for a lock.
type LockResponse struct {
	Lock Lock `json:"lock"`
	ErrorResponse
}
