package config

import (
	"bytes"
	"text/template"
)

var (
	configFileTmpl = template.Must(template.New("config").Parse(`# Soft Serve Server configurations

# The name of the server.
# This is the name that will be displayed in the UI.
name: "{{ .Name }}"

# The SSH server configuration.
ssh:
  # The address on which the SSH server will listen.
  listen_addr: "{{ .SSH.ListenAddr }}"

  # The public URL of the SSH server.
  # This is the address will be used to clone repositories.
  public_url: "{{ .SSH.PublicURL }}"

  # The relative path to the SSH server's private key.
  key_path: "{{ .SSH.KeyPath }}"

  # The relative path to the SSH server's client private key.
  # This key will be used to authenticate the server to make git requests to
  # ssh remotes.
  client_key_path: "{{ .SSH.ClientKeyPath }}"

  # The relative path to the SSH server's internal api private key.
  internal_key_path: "{{ .SSH.InternalKeyPath }}"

  # The maximum number of seconds a connection can take.
  # A value of 0 means no timeout.
  max_timeout: {{ .SSH.MaxTimeout }}

  # The number of seconds a connection can be idle before it is closed.
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
  # This is the address will be used to clone repositories.
  public_url: "{{ .HTTP.PublicURL }}"

# The stats server configuration.
stats:
  # The address on which the stats server will listen.
  listen_addr: "{{ .Stats.ListenAddr }}"

`))
)

func newConfigFile(cfg *Config) string {
	var b bytes.Buffer
	configFileTmpl.Execute(&b, cfg) // nolint: errcheck
	return b.String()
}
