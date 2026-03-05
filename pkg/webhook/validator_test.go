package webhook

import (
	"errors"
	"testing"

	"github.com/charmbracelet/soft-serve/pkg/ssrf"
)

// TestValidateWebhookURL verifies the wrapper delegates correctly and
// error aliases work across the package boundary. IP range coverage
// is in pkg/ssrf/ssrf_test.go -- here we just confirm the plumbing.
func TestValidateWebhookURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
		errType error
	}{
		{"valid", "https://1.1.1.1/webhook", false, nil},
		{"bad scheme", "ftp://example.com", true, ErrInvalidScheme},
		{"private IP", "http://127.0.0.1/webhook", true, ErrPrivateIP},
		{"empty", "", true, ErrInvalidURL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWebhookURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateWebhookURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errType != nil {
				if !errors.Is(err, tt.errType) {
					t.Errorf("ValidateWebhookURL(%q) error = %v, want %v", tt.url, err, tt.errType)
				}
			}
		})
	}
}

func TestErrorAliases(t *testing.T) {
	if ErrPrivateIP != ssrf.ErrPrivateIP {
		t.Error("ErrPrivateIP should alias ssrf.ErrPrivateIP")
	}
	if ErrInvalidScheme != ssrf.ErrInvalidScheme {
		t.Error("ErrInvalidScheme should alias ssrf.ErrInvalidScheme")
	}
	if ErrInvalidURL != ssrf.ErrInvalidURL {
		t.Error("ErrInvalidURL should alias ssrf.ErrInvalidURL")
	}
}
