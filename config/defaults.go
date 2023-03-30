package config

const defaultReadme = "# Soft Serve\n\n Welcome! You can configure your Soft Serve server by cloning this repo and pushing changes.\n\n```\ngit clone ssh://{{.Host}}:{{.Port}}/config\n```"

const defaultConfig = `# The name of the server to show in the TUI.
name: Soft Serve

# The host and port to display in the TUI. You may want to change this if your
# server is accessible from a different host and/or port that what it's
# actually listening on (for example, if it's behind a reverse proxy).
host: %s
port: %d

# Access level for anonymous users. Options are: admin-access, read-write,
# read-only, and no-access.
anon-access: %s

# You can grant read-only access to users without private keys. Any password
# will be accepted.
allow-keyless: %t

# You can select the order in which repositories are shown:
# 'alphabetical': alphabetical order
# 'commit': repositories with latest commits show first
# 'config': keep the order of config.yaml -> this is the default option
repos-order: config

# Customize repo display in the menu.
repos:
  - name: Home
    repo: config
    private: true
    note: "Configuration and content repo for this server"
    readme: README.md
`

const hasKeyUserConfig = `

# Authorized users. Admins have full access to all repos. Private repos are only
# accessible by admins and collab users. Regular users can read public repos
# based on your anon-access setting.
users:
  - name: Admin
    admin: true
    public-keys:
%s
`

const defaultUserConfig = `
# users:
#   - name: Admin
#     admin: true
#     public-keys:
#       - ssh-ed25519 AAAA... # redacted
#       - ssh-rsa AAAAB3Nz... # redacted`

const exampleUserConfig = `
#   - name: Example User
#     collab-repos:
#       - REPO
#     public-keys:
#       - ssh-ed25519 AAAA... # redacted
#       - ssh-rsa AAAAB3Nz... # redacted
`
