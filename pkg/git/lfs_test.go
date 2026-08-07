package git

import (
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/git-lfs-transfer/transfer"
	"github.com/charmbracelet/soft-serve/pkg/lfs"
)

// Object IDs arrive raw off the pktline stream. They must be rejected before
// they can be joined into a storage path. The backend is left zero-valued on
// purpose: a guard that fires only after the store or filesystem is touched is
// not a guard.
func TestLFSTransferRejectsMalformedOid(t *testing.T) {
	oids := []string{
		"../../../../../../../../etc/passwd",
		"../../ssh/soft_serve_host_ed25519",
		"objects/../../../soft-serve.db",
		"/etc/passwd",
		"",
		"abc",
		strings.Repeat("a", 63),
		strings.Repeat("a", 65),
		strings.ToUpper(strings.Repeat("a", 64)),
	}

	var backend lfsTransfer
	for _, oid := range oids {
		t.Run(oid, func(t *testing.T) {
			if _, _, err := backend.Download(oid, nil); !errors.Is(err, errInvalidOid) {
				t.Errorf("Download: got %v, want errInvalidOid", err)
			}
			if err := backend.Upload(oid, 1, strings.NewReader("x"), nil); !errors.Is(err, errInvalidOid) {
				t.Errorf("Upload: got %v, want errInvalidOid", err)
			}
			// Verify answers in-band with a conflict status rather than
			// tearing down the session.
			status, err := backend.Verify(oid, 1, nil)
			if err != nil {
				t.Errorf("Verify: unexpected error %v", err)
			} else if status == nil || status.Code() != transfer.StatusConflict {
				t.Errorf("Verify: got %v, want a conflict status", status)
			}

			items := []transfer.BatchItem{{Pointer: transfer.Pointer{Oid: oid, Size: 1}}}
			if _, err := backend.Batch("download", items, nil); !errors.Is(err, errInvalidOid) {
				t.Errorf("Batch: got %v, want errInvalidOid", err)
			}
		})
	}
}

// A malformed object ID is a client error, so the processor reports it as a
// 400 rather than falling through to a generic internal error.
func TestInvalidOidIsAParseError(t *testing.T) {
	if !errors.Is(errInvalidOid, transfer.ErrParseError) {
		t.Errorf("errInvalidOid does not wrap transfer.ErrParseError")
	}
	if !errors.Is(errInvalidOid, lfs.ErrInvalidOIDFormat) {
		t.Errorf("errInvalidOid does not wrap lfs.ErrInvalidOIDFormat")
	}
}
