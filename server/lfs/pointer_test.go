package lfs

import (
	"errors"
	"strconv"
	"strings"
	"testing"
)

func TestReadPointer(t *testing.T) {
	cases := []struct {
		name     string
		content  string
		want     Pointer
		wantErr  error
		wantErrp interface{}
	}{
		{
			name: "valid pointer",
			content: `version https://git-lfs.github.com/spec/v1
oid sha256:1234567890123456789012345678901234567890123456789012345678901234
size 1234
`,
			want: Pointer{
				Oid:  "1234567890123456789012345678901234567890123456789012345678901234",
				Size: 1234,
			},
		},
		{
			name: "invalid prefix",
			content: `version https://foobar/spec/v2
oid sha256:1234567890123456789012345678901234567890123456789012345678901234
size 1234
`,
			wantErr: ErrMissingPrefix,
		},
		{
			name: "invalid oid",
			content: `version https://git-lfs.github.com/spec/v1
oid sha256:&2345a78$012345678901234567890123456789012345678901234567890123
size 1234
`,
			wantErr: ErrInvalidOIDFormat,
		},
		{
			name: "invalid size",
			content: `version https://git-lfs.github.com/spec/v1
oid sha256:1234567890123456789012345678901234567890123456789012345678901234
size abc
`,
			wantErrp: &strconv.NumError{},
		},
		{
			name: "invalid structure",
			content: `version https://git-lfs.github.com/spec/v1
`,
			wantErr: ErrInvalidStructure,
		},
		{
			name:    "empty pointer",
			wantErr: ErrMissingPrefix,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, err := ReadPointerFromBuffer([]byte(tc.content))
			if err != tc.wantErr && !errors.As(err, &tc.wantErrp) {
				t.Errorf("ReadPointerFromBuffer() error = %v(%T), wantErr %v(%T)", err, err, tc.wantErr, tc.wantErr)
				return
			}
			if err != nil {
				return
			}

			if err == nil {
				if !p.IsValid() {
					t.Errorf("Expected a valid pointer")
					return
				}
				if p.Oid != strings.ReplaceAll(p.RelativePath(), "/", "") {
					t.Errorf("Expected oid to be the relative path without slashes")
					return
				}
			}

			if p.Oid != tc.want.Oid {
				t.Errorf("ReadPointerFromBuffer() oid = %v, want %v", p.Oid, tc.want.Oid)
			}
			if p.Size != tc.want.Size {
				t.Errorf("ReadPointerFromBuffer() size = %v, want %v", p.Size, tc.want.Size)
			}
		})
	}
}
