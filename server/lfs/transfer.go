package lfs

import (
	"context"
	"io"
)

// TransferBasic is the name of the Git LFS basic transfer protocol.
const TransferBasic = "basic"

// TransferAdapter represents an adapter for downloading/uploading LFS objects
type TransferAdapter interface {
	Name() string
	Download(ctx context.Context, p Pointer, l *Link) (io.ReadCloser, error)
	Upload(ctx context.Context, p Pointer, r io.Reader, l *Link) error
	Verify(ctx context.Context, p Pointer, l *Link) error
}
