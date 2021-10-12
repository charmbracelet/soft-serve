package config

const defaultReadme = "# Soft Serve\n\n Welcome! You can configure your Soft Serve server by cloning this repo and pushing changes.\n\n```\ngit clone ssh://{{.Host}}:{{.Port}}/config\n```"

const defaultConfig = `name: Soft Serve
host: %s
port: %d

# Set the access level for anonymous users. Options are: read-write, read-only and no-access.
anon-access: %s

# Grant read-only access to users without private keys. Any password will be accepted.
allow-keyless: false

# Customize repo display in the menu.
repos:
  - name: Home
    repo: config
    private: true
    note: "Configuration and content repo for this server"`

const hasKeyUserConfig = `

# Users can read all repos and push to collab-repos. Admins have full access to all repos.
users:
  - name: admin
    admin: true
    public-keys:
%s`

const defaultUserConfig = `
# users:
#   - name: admin
#     admin: true
#     public-keys:
#       - KEY TEXT`

const exampleUserConfig = `
#   - name: Example User
#     collab-repos:
#       - REPO
#     public-keys:
#       - KEY TEXT`
