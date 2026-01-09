package lfs

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"strconv"
	"strings"
)

const (
	blobSizeCutoff = 1024

	// HashAlgorithmSHA256 is the hash algorithm used for Git LFS.
	HashAlgorithmSHA256 = "sha256"

	// MetaFileIdentifier is the string appearing at the first line of LFS pointer files.
	// https://github.com/git-lfs/git-lfs/blob/master/docs/spec.md
	MetaFileIdentifier = "version https://git-lfs.github.com/spec/v1"

	// MetaFileOidPrefix appears in LFS pointer files on a line before the sha256 hash.
	MetaFileOidPrefix = "oid " + HashAlgorithmSHA256 + ":"
)

var (
	// ErrMissingPrefix occurs if the content lacks the LFS prefix
	ErrMissingPrefix = errors.New("content lacks the LFS prefix")

	// ErrInvalidStructure occurs if the content has an invalid structure
	ErrInvalidStructure = errors.New("content has an invalid structure")

	// ErrInvalidOIDFormat occurs if the oid has an invalid format
	ErrInvalidOIDFormat = errors.New("OID has an invalid format")
)

// ReadPointer tries to read LFS pointer data from the reader
func ReadPointer(reader io.Reader) (Pointer, error) {
	buf := make([]byte, blobSizeCutoff)
	n, err := io.ReadFull(reader, buf)
	if err != nil && err != io.ErrUnexpectedEOF {
		return Pointer{}, err
	}
	buf = buf[:n]

	return ReadPointerFromBuffer(buf)
}

var oidPattern = regexp.MustCompile(`^[a-f\d]{64}$`)

// ReadPointerFromBuffer will return a pointer if the provided byte slice is a pointer file or an error otherwise.
func ReadPointerFromBuffer(buf []byte) (Pointer, error) {
	var p Pointer

	headString := string(buf)
	if !strings.HasPrefix(headString, MetaFileIdentifier) {
		return p, ErrMissingPrefix
	}

	splitLines := strings.Split(headString, "\n")
	if len(splitLines) < 3 {
		return p, ErrInvalidStructure
	}

	oid := strings.TrimPrefix(splitLines[1], MetaFileOidPrefix)
	if len(oid) != 64 || !oidPattern.MatchString(oid) {
		return p, ErrInvalidOIDFormat
	}
	size, err := strconv.ParseInt(strings.TrimPrefix(splitLines[2], "size "), 10, 64)
	if err != nil {
		return p, err
	}

	p.Oid = oid
	p.Size = size

	return p, nil
}

// IsValid checks if the pointer has a valid structure.
// It doesn't check if the pointed-to-content exists.
func (p Pointer) IsValid() bool {
	if len(p.Oid) != 64 {
		return false
	}
	if !oidPattern.MatchString(p.Oid) {
		return false
	}
	if p.Size < 0 {
		return false
	}
	return true
}

// String returns the string representation of the pointer
// https://github.com/git-lfs/git-lfs/blob/main/docs/spec.md#the-pointer
func (p Pointer) String() string {
	return fmt.Sprintf("%s\n%s%s\nsize %d\n", MetaFileIdentifier, MetaFileOidPrefix, p.Oid, p.Size)
}

// RelativePath returns the relative storage path of the pointer
// https://github.com/git-lfs/git-lfs/blob/main/docs/spec.md#intercepting-git
func (p Pointer) RelativePath() string {
	if len(p.Oid) < 5 {
		return p.Oid
	}

	return path.Join(p.Oid[0:2], p.Oid[2:4], p.Oid)
}

// GeneratePointer generates a pointer for arbitrary content
func GeneratePointer(content io.Reader) (Pointer, error) {
	h := sha256.New()
	c, err := io.Copy(h, content)
	if err != nil {
		return Pointer{}, err
	}
	sum := h.Sum(nil)
	return Pointer{Oid: hex.EncodeToString(sum), Size: c}, nil
}
