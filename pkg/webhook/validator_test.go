package webhook

import (
	"net"
	"testing"
)

func TestValidateWebhookURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
		errType error
		skip    string
	}{
		// Valid URLs (these will perform DNS lookups, so may fail in some environments)
		{
			name:    "valid https URL",
			url:     "https://1.1.1.1/webhook",
			wantErr: false,
		},
		{
			name:    "valid http URL",
			url:     "http://8.8.8.8/webhook",
			wantErr: false,
		},
		{
			name:    "valid URL with port",
			url:     "https://1.1.1.1:8080/webhook",
			wantErr: false,
		},
		{
			name:    "valid URL with path and query",
			url:     "https://8.8.8.8/webhook?token=abc123",
			wantErr: false,
		},

		// Invalid schemes
		{
			name:    "ftp scheme",
			url:     "ftp://example.com/webhook",
			wantErr: true,
			errType: ErrInvalidScheme,
		},
		{
			name:    "file scheme",
			url:     "file:///etc/passwd",
			wantErr: true,
			errType: ErrInvalidScheme,
		},
		{
			name:    "gopher scheme",
			url:     "gopher://example.com",
			wantErr: true,
			errType: ErrInvalidScheme,
		},
		{
			name:    "no scheme",
			url:     "example.com/webhook",
			wantErr: true,
			errType: ErrInvalidScheme,
		},

		// Localhost variations
		{
			name:    "localhost",
			url:     "http://localhost/webhook",
			wantErr: true,
			errType: ErrPrivateIP,
		},
		{
			name:    "localhost with port",
			url:     "http://localhost:8080/webhook",
			wantErr: true,
			errType: ErrPrivateIP,
		},
		{
			name:    "localhost.localdomain",
			url:     "http://localhost.localdomain/webhook",
			wantErr: true,
			errType: ErrPrivateIP,
		},

		// Loopback IPs
		{
			name:    "127.0.0.1",
			url:     "http://127.0.0.1/webhook",
			wantErr: true,
			errType: ErrPrivateIP,
		},
		{
			name:    "127.0.0.1 with port",
			url:     "http://127.0.0.1:8080/webhook",
			wantErr: true,
			errType: ErrPrivateIP,
		},
		{
			name:    "127.1.2.3",
			url:     "http://127.1.2.3/webhook",
			wantErr: true,
			errType: ErrPrivateIP,
		},
		{
			name:    "IPv6 loopback",
			url:     "http://[::1]/webhook",
			wantErr: true,
			errType: ErrPrivateIP,
		},

		// Private IPv4 ranges
		{
			name:    "10.0.0.0",
			url:     "http://10.0.0.1/webhook",
			wantErr: true,
			errType: ErrPrivateIP,
		},
		{
			name:    "192.168.0.0",
			url:     "http://192.168.1.1/webhook",
			wantErr: true,
			errType: ErrPrivateIP,
		},
		{
			name:    "172.16.0.0",
			url:     "http://172.16.0.1/webhook",
			wantErr: true,
			errType: ErrPrivateIP,
		},
		{
			name:    "172.31.255.255",
			url:     "http://172.31.255.255/webhook",
			wantErr: true,
			errType: ErrPrivateIP,
		},

		// Link-local (AWS/GCP/Azure metadata)
		{
			name:    "AWS metadata service",
			url:     "http://169.254.169.254/latest/meta-data/",
			wantErr: true,
			errType: ErrPrivateIP,
		},
		{
			name:    "link-local",
			url:     "http://169.254.1.1/webhook",
			wantErr: true,
			errType: ErrPrivateIP,
		},

		// Other reserved ranges
		{
			name:    "0.0.0.0",
			url:     "http://0.0.0.0/webhook",
			wantErr: true,
			errType: ErrPrivateIP,
		},
		{
			name:    "broadcast",
			url:     "http://255.255.255.255/webhook",
			wantErr: true,
			errType: ErrPrivateIP,
		},

		// Invalid URLs
		{
			name:    "empty URL",
			url:     "",
			wantErr: true,
			errType: ErrInvalidURL,
		},
		{
			name:    "missing hostname",
			url:     "http:///webhook",
			wantErr: true,
			errType: ErrInvalidURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip != "" {
				t.Skip(tt.skip)
			}
			err := ValidateWebhookURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateWebhookURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errType != nil {
				if !isErrorType(err, tt.errType) {
					t.Errorf("ValidateWebhookURL() error = %v, want error type %v", err, tt.errType)
				}
			}
		})
	}
}

func TestIsPrivateOrInternalIP(t *testing.T) {
	tests := []struct {
		name   string
		ip     string
		isPriv bool
	}{
		// Public IPs
		{"Google DNS", "8.8.8.8", false},
		{"Cloudflare DNS", "1.1.1.1", false},
		{"Public IPv6", "2001:4860:4860::8888", false},

		// Loopback
		{"127.0.0.1", "127.0.0.1", true},
		{"127.1.2.3", "127.1.2.3", true},
		{"::1", "::1", true},

		// Private ranges
		{"10.0.0.1", "10.0.0.1", true},
		{"192.168.1.1", "192.168.1.1", true},
		{"172.16.0.1", "172.16.0.1", true},
		{"172.31.255.255", "172.31.255.255", true},

		// Link-local
		{"169.254.169.254", "169.254.169.254", true},
		{"169.254.1.1", "169.254.1.1", true},
		{"fe80::1", "fe80::1", true},

		// Other reserved
		{"0.0.0.0", "0.0.0.0", true},
		{"255.255.255.255", "255.255.255.255", true},
		{"240.0.0.1", "240.0.0.1", true},

		// Shared address space
		{"100.64.0.1", "100.64.0.1", true},
		{"100.127.255.255", "100.127.255.255", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("Failed to parse IP: %s", tt.ip)
			}
			if got := isPrivateOrInternalIP(ip); got != tt.isPriv {
				t.Errorf("isPrivateOrInternalIP(%s) = %v, want %v", tt.ip, got, tt.isPriv)
			}
		})
	}
}

func TestIsLocalhost(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		want     bool
	}{
		{"localhost", "localhost", true},
		{"LOCALHOST", "LOCALHOST", true},
		{"localhost.localdomain", "localhost.localdomain", true},
		{"test.localhost", "test.localhost", true},
		{"example.com", "example.com", false},
		{"localhos", "localhos", false},
		{"localhost.com", "localhost.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isLocalhost(tt.hostname); got != tt.want {
				t.Errorf("isLocalhost(%s) = %v, want %v", tt.hostname, got, tt.want)
			}
		})
	}
}

func TestValidateIPBeforeDial(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		wantErr bool
	}{
		{"public IP", "8.8.8.8", false},
		{"private IP", "192.168.1.1", true},
		{"loopback", "127.0.0.1", true},
		{"link-local", "169.254.169.254", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("Failed to parse IP: %s", tt.ip)
			}
			err := ValidateIPBeforeDial(ip)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIPBeforeDial(%s) error = %v, wantErr %v", tt.ip, err, tt.wantErr)
			}
		})
	}
}

// isErrorType checks if err is or wraps errType.
func isErrorType(err, errType error) bool {
	if err == errType {
		return true
	}
	// Check if err wraps errType
	for err != nil {
		if err == errType {
			return true
		}
		unwrapped, ok := err.(interface{ Unwrap() error })
		if !ok {
			break
		}
		err = unwrapped.Unwrap()
	}
	return false
}
