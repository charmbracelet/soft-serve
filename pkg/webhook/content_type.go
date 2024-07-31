package webhook

import (
	"encoding"
	"errors"
	"strings"
)

// ContentType is the type of content that will be sent in a webhook request.
type ContentType int8

const (
	// ContentTypeJSON is the JSON content type.
	ContentTypeJSON ContentType = iota
	// ContentTypeForm is the form content type.
	ContentTypeForm
)

var contentTypeStrings = map[ContentType]string{
	ContentTypeJSON: "application/json",
	ContentTypeForm: "application/x-www-form-urlencoded",
}

// String returns the string representation of the content type.
func (c ContentType) String() string {
	return contentTypeStrings[c]
}

var stringContentType = map[string]ContentType{
	"application/json":                  ContentTypeJSON,
	"application/x-www-form-urlencoded": ContentTypeForm,
}

// ErrInvalidContentType is returned when the content type is invalid.
var ErrInvalidContentType = errors.New("invalid content type")

// ParseContentType parses a content type string and returns the content type.
func ParseContentType(s string) (ContentType, error) {
	for k, v := range stringContentType {
		if strings.HasPrefix(s, k) {
			return v, nil
		}
	}

	return -1, ErrInvalidContentType
}

var (
	_ encoding.TextMarshaler   = ContentType(0)
	_ encoding.TextUnmarshaler = (*ContentType)(nil)
)

// UnmarshalText implements encoding.TextUnmarshaler.
func (c *ContentType) UnmarshalText(text []byte) error {
	ct, err := ParseContentType(string(text))
	if err != nil {
		return err
	}

	*c = ct
	return nil
}

// MarshalText implements encoding.TextMarshaler.
func (c ContentType) MarshalText() (text []byte, err error) {
	ct := c.String()
	if ct == "" {
		return nil, ErrInvalidContentType
	}

	return []byte(ct), nil
}
