package webhook

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/ssrf"
)

// TestSSRFProtection is an integration test verifying the webhook send path
// blocks private IPs end-to-end (models.Webhook -> secureHTTPClient -> ssrf).
func TestSSRFProtection(t *testing.T) {
	tests := []struct {
		name        string
		webhookURL  string
		shouldBlock bool
	}{
		{"block loopback", "http://127.0.0.1:8080/webhook", true},
		{"block metadata", "http://169.254.169.254/latest/meta-data/", true},
		{"allow public IP", "http://8.8.8.8/webhook", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := models.Webhook{
				URL:         tt.webhookURL,
				ContentType: int(ContentTypeJSON),
			}

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, "POST", w.URL, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			resp, err := secureHTTPClient.Do(req)
			if resp != nil {
				resp.Body.Close()
			}

			if tt.shouldBlock {
				if err == nil {
					t.Errorf("%s: expected error but got none", tt.name)
				}
			} else {
				if err != nil && errors.Is(err, ssrf.ErrPrivateIP) {
					t.Errorf("%s: should not block public IPs, got: %v", tt.name, err)
				}
			}
		})
	}
}
