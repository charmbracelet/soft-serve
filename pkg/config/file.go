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
  # Enable SSH.
  enabled: {{ .SSH.Enabled }}

  # The address on which the SSH server will listen.
  listen_addr: "{{ .SSH.ListenAddr }}"

  # The public URL of the SSH server.
  # This is the address that will be used to clone repositories.
  public_url: "{{ .SSH.PublicURL }}"

  # The path to the SSH server's private key.
  key_path: {{ .SSH.KeyPath }}

  # The path to the server's client private key. This key will be used to
  # authenticate the server to make git requests to ssh remotes.
  client_key_path: {{ .SSH.ClientKeyPath }}

  # The maximum number of seconds a connection can take.
  # A value of 0 means no timeout.
  max_timeout: {{ .SSH.MaxTimeout }}

  # The number of seconds a connection can be idle before it is closed.
  # A value of 0 means no timeout.
  idle_timeout: {{ .SSH.IdleTimeout }}

  # Allow mouse events in the TUI. When true (default), mouse clicks and
  # scrolling work in the TUI. Set to false to restore terminal text selection.
  # Can also be controlled with SOFT_SERVE_SSH_ALLOW_MOUSE_EVENTS=false.
  allow_mouse_events: {{ .SSH.AllowMouseEvents }}

  # Allowed SSH key exchange algorithms.
  # Leave empty to use the go/crypto defaults.
  # key_exchanges:
  #   - curve25519-sha256
  #   - ecdh-sha2-nistp256

  # Allowed SSH ciphers.
  # Leave empty to use the go/crypto defaults.
  # ciphers:
  #   - aes128-gcm@openssh.com
  #   - chacha20-poly1305@openssh.com

  # Allowed SSH MAC algorithms.
  # Leave empty to use the go/crypto defaults.
  # macs:
  #   - hmac-sha2-256-etm@openssh.com

# The Git daemon configuration.
git:
  # Enable the Git daemon.
  enabled: {{ .Git.Enabled }}

  # The address on which the Git daemon will listen.
  listen_addr: "{{ .Git.ListenAddr }}"

  # The public URL of the Git daemon server.
  # This is the address that will be used to clone repositories.
  public_url: "{{ .Git.PublicURL }}"

  # The maximum number of seconds a connection can take.
  # A value of 0 means no timeout.
  max_timeout: {{ .Git.MaxTimeout }}

  # The number of seconds a connection can be idle before it is closed.
  idle_timeout: {{ .Git.IdleTimeout }}

  # The maximum number of concurrent connections.
  max_connections: {{ .Git.MaxConnections }}

# The HTTP server configuration.
http:
  # Enable the HTTP server.
  enabled: {{ .HTTP.Enabled }}

  # The address on which the HTTP server will listen.
  listen_addr: "{{ .HTTP.ListenAddr }}"

  # The path to the TLS private key.
  tls_key_path: {{ .HTTP.TLSKeyPath }}

  # The path to the TLS certificate.
  tls_cert_path: {{ .HTTP.TLSCertPath }}

  # The public URL of the HTTP server.
  # This is the address that will be used to clone repositories.
  # Make sure to use https:// if you are using TLS.
  public_url: "{{ .HTTP.PublicURL }}"

  # When true, repositories are accessible without the .git suffix in the URL.
  # Both /<name> and /<name>.git will be accepted.
  # strip_git_suffix: false

  # When true, the X-Forwarded-For header is trusted for client IP resolution.
  # Only enable this when the server sits behind a trusted reverse proxy.
  # trust_proxy_headers: false

  # Maximum HTTP requests per second per IP address. Set to 0 to disable.
  # rate_limit: 10

  # Maximum burst size for the HTTP rate limiter.
  # rate_burst: 30

  # The cross-origin request security options
  cors:
    # The allowed cross-origin headers
    allowed_headers:
       - "Accept"
       - "Accept-Language"
       - "Content-Language"
       - "Content-Type"
       - "Origin"
       - "X-Requested-With"
       - "User-Agent"
       - "Authorization"
       - "Access-Control-Request-Method"
       - "Access-Control-Allow-Origin"
    # The allowed cross-origin URLs
    allowed_origins:
       - "{{ .HTTP.PublicURL }}" # always allowed
       # - "https://example.com"
    # The allowed cross-origin methods
    allowed_methods:
       - "GET"
       - "HEAD"
       - "POST"
       - "PUT"
       - "OPTIONS"

# The stats server configuration.
stats:
  # Enable the stats server.
  enabled: {{ .Stats.Enabled }}

  # The address on which the stats server will listen.
  listen_addr: "{{ .Stats.ListenAddr }}"

# The database configuration.
db:
  # The database driver to use.
  # Valid values are "sqlite" and "postgres".
  driver: "{{ .DB.Driver }}"
  # The database data source name.
  # This is driver specific and can be a file path or connection string.
  # Make sure foreign key support is enabled when using SQLite.
  data_source: "{{ .DB.DataSource }}"

# Git LFS configuration.
lfs:
  # Enable Git LFS.
  enabled: {{ .LFS.Enabled }}
  # Enable Git SSH transfer.
  ssh_enabled: {{ .LFS.SSHEnabled }}

# Cron job configuration
jobs:
  mirror_pull:
    # Enable the periodic mirror pull job.
    enabled: {{ .Jobs.MirrorPull.Enabled }}
    # Cron schedule for the mirror pull job.
    schedule: "{{ .Jobs.MirrorPull.Schedule }}"

# Additional admin keys.
#initial_admin_keys:
#  - "ssh-rsa AAAAB3NzaC1yc2..."

# Anonymous access level applied on every startup.
# Overrides the value stored in the database when set.
# Valid values: no-access, read-only, read-write, admin-access.
# Leave commented out to preserve the database value.
#anon_access: read-only

# Whether keyless (keyboard-interactive) access is allowed.
# Overrides the value stored in the database when set.
# Leave commented out to preserve the database value.
#allow_keyless: true

# When true, serve go-get meta tags for private/hidden repositories.
# The actual git content remains inaccessible without credentials.
# allow_public_go_get: false
`))

func newConfigFile(cfg *Config) string {
	var b bytes.Buffer
	configFileTmpl.Execute(&b, cfg) //nolint: errcheck
	return b.String()
}
