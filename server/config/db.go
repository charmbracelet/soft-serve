package config

// DatabaseConfig is the database configuration.
type DatabaseConfig struct {
	// Driver is the database driver.
	Driver string `env:"DRIVER" yaml:"driver"`

	// DataSource is the database data source.
	DataSource string `env:"DATA_SOURCE" yaml:"data_source"`
}

// Environ returns the environment variables for the database configuration.
func (d DatabaseConfig) Environ() []string {
	envs := []string{
		"SOFT_SERVE_DATABASE_DRIVER=" + d.Driver,
		"SOFT_SERVE_DATABASE_DATA_SOURCE=" + d.DataSource,
	}

	return envs
}
