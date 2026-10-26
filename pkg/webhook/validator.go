package webhook

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/ssrf"
)

// Error aliases for backward compatibility.
var (
	ErrInvalidScheme = ssrf.ErrInvalidScheme
	ErrPrivateIP     = ssrf.ErrPrivateIP
	ErrInvalidURL    = ssrf.ErrInvalidURL
)

// ValidateWebhookURL validates that a webhook URL is safe to use.
func ValidateWebhookURL(rawURL string, opts ...interface{}) error {
	pass := make([]interface{}, 0, len(opts))
	for _, o := range opts {
		if ctx, ok := o.(context.Context); ok {
			if cfg := config.FromContext(ctx); cfg != nil {
				pass = append(pass, cfg.InsecureInternalServers)
			}
			continue
		}
		pass = append(pass, o)
	}
	return ssrf.ValidateURL(rawURL, pass...) //nolint:wrapcheck
}

// ValidateIPBeforeDial validates an IP address before establishing a connection.
var ValidateIPBeforeDial = ssrf.ValidateIPBeforeDial