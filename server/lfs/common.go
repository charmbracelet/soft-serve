package lfs

import "time"

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
