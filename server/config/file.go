package config

import (
	"bytes"
	"text/template"
)

var configFileTmpl = template.Must(template.New("config").Parse(`# Soft Serve Server configurations

# The name of the server.
# This is the name that will be displayed in the UI.
name: "{{ .Name }}"

# Logging configuration.
log:
  # Log format to use. Valid values are "json", "logfmt", and "text".
  format: "{{ .Log.Format }}"
  # Time format for the log "timestamp" field.
  # Should be described in Golang's time format.
  time_format: "{{ .Log.TimeFormat }}"
  # Path to the log file. Leave empty to write to stderr.
  #path: "{{ .Log.Path }}"

# The SSH server configuration.
ssh:
  # The address on which the SSH server will listen.
  listen_addr: "{{ .SSH.ListenAddr }}"

  # The public URL of the SSH server.
  # This is the address that will be used to clone repositories.
  public_url: "{{ .SSH.PublicURL }}"

  # The path to the SSH server's private key.
  key_path: "{{ .SSH.KeyPath }}"

  # The path to the server's client private key. This key will be used to
  # authenticate the server to make git requests to ssh remotes.
  client_key_path: "{{ .SSH.ClientKeyPath }}"

  # The maximum number of seconds a connection can take.
  # A value of 0 means no timeout.
  max_timeout: {{ .SSH.MaxTimeout }}

  # The number of seconds a connection can be idle before it is closed.
  # A value of 0 means no timeout.
  idle_timeout: {{ .SSH.IdleTimeout }}

# The Git daemon configuration.
git:
  # The address on which the Git daemon will listen.
  listen_addr: "{{ .Git.ListenAddr }}"

  # The maximum number of seconds a connection can take.
  # A value of 0 means no timeout.
  max_timeout: {{ .Git.MaxTimeout }}

  # The number of seconds a connection can be idle before it is closed.
  idle_timeout: {{ .Git.IdleTimeout }}

  # The maximum number of concurrent connections.
  max_connections: {{ .Git.MaxConnections }}

# The HTTP server configuration.
http:
  # The address on which the HTTP server will listen.
  listen_addr: "{{ .HTTP.ListenAddr }}"

  # The path to the TLS private key.
  tls_key_path: "{{ .HTTP.TLSKeyPath }}"

  # The path to the TLS certificate.
  tls_cert_path: "{{ .HTTP.TLSCertPath }}"

  # The public URL of the HTTP server.
  # This is the address that will be used to clone repositories.
  # Make sure to use https:// if you are using TLS.
  public_url: "{{ .HTTP.PublicURL }}"

# The stats server configuration.
stats:
  # The address on which the stats server will listen.
  listen_addr: "{{ .Stats.ListenAddr }}"

# The database configuration.
db:
  # The database driver to use.
  # Valid values are "sqlite" and "postgres".
  driver: "{{ .DB.Driver }}"
  # The database data source name.
  # This is driver specific and can be a file path or connection string.
  data_source: "{{ .DB.DataSource }}"

# Git LFS configuration.
lfs:
  # Enable Git LFS.
  enabled: {{ .LFS.Enabled }}
  # Enable Git SSH transfer.
  ssh_enabled: {{ .LFS.SSHEnabled }}

# Additional admin keys.
#initial_admin_keys:
#  - "ssh-rsa AAAAB3NzaC1yc2..."
`))

func newConfigFile(cfg *Config) string {
	var b bytes.Buffer
	configFileTmpl.Execute(&b, cfg) // nolint: errcheck
	return b.String()
}
