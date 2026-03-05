package webhook

import (
	"github.com/charmbracelet/soft-serve/pkg/ssrf"
)

// Error aliases for backward compatibility.
var (
	ErrInvalidScheme = ssrf.ErrInvalidScheme
	ErrPrivateIP     = ssrf.ErrPrivateIP
	ErrInvalidURL    = ssrf.ErrInvalidURL
)

// ValidateWebhookURL validates that a webhook URL is safe to use.
func ValidateWebhookURL(rawURL string) error {
	return ssrf.ValidateURL(rawURL) //nolint:wrapcheck
}

// ValidateIPBeforeDial validates an IP address before establishing a connection.
var ValidateIPBeforeDial = ssrf.ValidateIPBeforeDial
