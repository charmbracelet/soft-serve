package config

// BackendConfig is the backend configuration.
type BackendConfig struct {
	// Settings is the settings backend.
	Settings string `env:"SETTINGS" yaml:"settings"`

	// Access is the access backend.
	Access string `env:"ACCESS" yaml:"access"`

	// Store is the store backend.
	Store string `env:"STORE" yaml:"store"`

	// Auth is the auth backend.
	Auth string `env:"AUTH" yaml:"auth"`
}

// Environ returns the environment variables for the backend configuration.
func (b BackendConfig) Environ() []string {
	envs := []string{
		"SOFT_SERVE_BACKEND_SETTINGS=" + b.Settings,
		"SOFT_SERVE_BACKEND_ACCESS=" + b.Access,
		"SOFT_SERVE_BACKEND_STORE=" + b.Store,
		"SOFT_SERVE_BACKEND_AUTH=" + b.Auth,
	}

	return envs
}
