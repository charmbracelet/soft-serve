package lfs

import (
	"context"
	"io"
)

const (
	// SchemeHTTP represents the HTTP protocol scheme
	SchemeHTTP = "http"
	// SchemeHTTPS represents the HTTPS protocol scheme
	SchemeHTTPS = "https"
)

// DownloadCallback gets called for every requested LFS object to process its content
type DownloadCallback func(p Pointer, content io.ReadCloser, objectError error) error

// UploadCallback gets called for every requested LFS object to provide its content
type UploadCallback func(p Pointer, objectError error) (io.ReadCloser, error)

// Client is a Git LFS client to communicate with a LFS source API.
type Client interface {
	Download(ctx context.Context, objects []Pointer, callback DownloadCallback) error
	Upload(ctx context.Context, objects []Pointer, callback UploadCallback) error
}

// NewClient returns a new Git LFS client.
func NewClient(e Endpoint) Client {
	if e.Scheme == SchemeHTTP || e.Scheme == SchemeHTTPS {
		return newHTTPClient(e)
	}
	// TODO: support ssh client
	return nil
}
