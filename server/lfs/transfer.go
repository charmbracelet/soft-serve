package lfs

import (
	"context"
	"io"
)

// TransferAdapter represents an adapter for downloading/uploading LFS objects
type TransferAdapter interface {
	Name() string
	Download(ctx context.Context, p Pointer, l *Link) (io.ReadCloser, error)
	Upload(ctx context.Context, p Pointer, r io.Reader, l *Link) error
	Verify(ctx context.Context, p Pointer, l *Link) error
}
