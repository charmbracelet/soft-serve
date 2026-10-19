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

  # How the server's ssh client verifies host keys of ssh remotes it imports
  # or mirrors from, checked against <data_path>/ssh/known_hosts.
  # "accept-new" records a host key on first contact and refuses a changed
  # one. "yes" refuses any host that is not already in known_hosts; use it
  # with a pre-seeded known_hosts file. The client always runs with
  # BatchMode=yes, so it fails fast instead of prompting for a passphrase.
  client_strict_host_key_checking: "{{ .SSH.ClientStrictHostKeyChecking }}"

  # The ssh hosts that repositories may be imported from or mirrored. The
  # client key above is presented to every host a user imports from, so list
  # only hosts you trust that key with. Entries are hostnames, IP addresses,
  # or CIDR ranges, each with an optional port. When the list is empty, only
  # server admins may import from or mirror ssh remotes.
  # CIDR entries are checked against DNS only at import or sync time and ssh
  # resolves the name again, so prefer hostnames for anything you do not
  # control. The list is matched against the name given to ssh; your
  # ssh_config still applies to it.
  client_allowed_hosts:
{{- range .SSH.ClientAllowedHosts }}
    - "{{ . }}"
{{- else }} []
    # - "git.internal.example"
    # - "gitlab.corp:2222"
    # - "10.0.0.0/8"
{{- end }}

  # The maximum number of seconds a connection can take.
  # A value of 0 means no timeout.
  max_timeout: {{ .SSH.MaxTimeout }}

  # The number of seconds a connection can be idle before it is closed.
  # A value of 0 means no timeout.
  idle_timeout: {{ .SSH.IdleTimeout }}

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
  mirror_pull: "{{ .Jobs.MirrorPull }}"

# Additional admin keys.
#initial_admin_keys:
#  - "ssh-rsa AAAAB3NzaC1yc2..."

# Name of a repository to create automatically on boot if it doesn't already
# exist. Empty disables this behavior.
#default_repo: ""
`))

func newConfigFile(cfg *Config) string {
	var b bytes.Buffer
	configFileTmpl.Execute(&b, cfg) //nolint: errcheck
	return b.String()
}
