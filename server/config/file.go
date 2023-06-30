package config

import (
	"bytes"
	"text/template"
)

var configFileTmpl = template.Must(template.New("config").Parse(`# Soft Serve Server configurations

# The name of the server.
# This is the name that will be displayed in the UI.
name: "{{ .Name }}"

# Cache configuration.
cache:
  # The cache backend to use. The default backend is "lru" memory cache.
  backend: "{{ .Cache.Backend }}"

# Database configuration.
database:
  # The database driver to use. The default driver is "sqlite".
  driver: "{{ .Database.Driver }}"
  # The data source to the database. For sqlite, this is the path to the
  # database file.
  data_source:"{{ .Database.DataSource }}"

# Backend configuration defines the backend for server settings, repositories,
# authentication, and authorization.
backend:
  # Settings is the server settings backend.
  # The default is "sqlite" which stores server settings as a key-value pair in
  # the database.
  settings: "{{ .Backend.Settings }}"
  # Access is the authorization backend.
  # The default is "sqlite" which stores access rules in the database.
  access: "{{ .Backend.Access }}"
  # Auth is the authentication backend.
  # The default is "sqlite" which stores and manages users in the database.
  auth: "{{ .Backend.Auth }}"
  # Store is the repository storage backend.
  # The default is "filesqlite" which stores repositories in the filesystem and
  # the sqlite database.
  # Git repositories are stored in the filesystem and their metadata are stored
  # in both the filesystem and database.
  store: "{{ .Backend.Store }}"

# Logging configuration.
log:
  # Log format to use. Valid values are "json", "logfmt", and "text".
  format: "{{ .Log.Format }}"
  # Time format for the log "timestamp" field.
  # Should be described in Golang's time format.
  time_format: "{{ .Log.TimeFormat }}"

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
git_daemon:
  # The address on which the Git daemon will listen.
  listen_addr: "{{ .GitDaemon.ListenAddr }}"

  # The maximum number of seconds a connection can take.
  # A value of 0 means no timeout.
  max_timeout: {{ .GitDaemon.MaxTimeout }}

  # The number of seconds a connection can be idle before it is closed.
  idle_timeout: {{ .GitDaemon.IdleTimeout }}

  # The maximum number of concurrent connections.
  max_connections: {{ .GitDaemon.MaxConnections }}

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
  # Note that by default, the stats server binds to "localhost".
  # This won't make it accessible from other networks.
  # If you're running Soft Serve on a container, you probably want it to be
  # accessible to other networks. To do so, change the listen address to
  # ":PORT" or "0.0.0.0:PORT".
  listen_addr: "{{ .Stats.ListenAddr }}"

# Additional admin keys.
#initial_admin_keys:
#  - "ssh-rsa AAAAB3NzaC1yc2..."
`))

func newConfigFile(cfg *Config) string {
	var b bytes.Buffer
	configFileTmpl.Execute(&b, cfg) // nolint: errcheck
	return b.String()
}
