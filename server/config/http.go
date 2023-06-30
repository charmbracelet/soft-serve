package config

// HTTPConfig is the HTTP configuration for the server.
type HTTPConfig struct {
	// ListenAddr is the address on which the HTTP server will listen.
	ListenAddr string `env:"LISTEN_ADDR" yaml:"listen_addr"`

	// TLSKeyPath is the path to the TLS private key.
	TLSKeyPath string `env:"TLS_KEY_PATH" yaml:"tls_key_path"`

	// TLSCertPath is the path to the TLS certificate.
	TLSCertPath string `env:"TLS_CERT_PATH" yaml:"tls_cert_path"`

	// PublicURL is the public URL of the HTTP server.
	PublicURL string `env:"PUBLIC_URL" yaml:"public_url"`
}

// Environ returns the environment variables for the config.
func (h HTTPConfig) Environ() []string {
	return []string{
		"SOFT_SERVE_HTTP_LISTEN_ADDR=" + h.ListenAddr,
		"SOFT_SERVE_HTTP_TLS_KEY_PATH=" + h.TLSKeyPath,
		"SOFT_SERVE_HTTP_TLS_CERT_PATH=" + h.TLSCertPath,
		"SOFT_SERVE_HTTP_PUBLIC_URL=" + h.PublicURL,
	}
}
