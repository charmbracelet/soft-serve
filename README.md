Soft Serve
==========

A tasty Git server that runs its own SSH service. 🍦

* Configure with `git`
* Create repos on demand with `git push`
* Browse repos with an SSH-accessible TUI
* Easy access control
  - Allow/disallow anonymous access
  - Add collaborators with SSH public keys
  - Repos can be public or private

## What does it look like?

Just run `ssh beta.charm.sh` for an example.

## Building/installing

The Soft Serve command is called `soft`. You can build and install it with
`go`:

```bash
git clone ssh://beta.charm.sh/soft-serve
cd soft-serve
go install
```

## Setting up a server

Make sure `git` is installed, then run `soft`. That’s it.

## Configuration

The Soft Serve configuration is simple and straightforward:

```yaml
# The name of the server to show in the TUI.
name: Soft Serve

# The host and port to listen on. Defaults to 0.0.0.0:23231.
host: localhost
port: 23231

# The access level for anonymous users. Options are: read-write, read-only
# and no-access.
anon-access: read-write

# You can grant read-only access to users without private keys.
allow-keyless: false

# Which repos should appear in the menu?
repos:
  - name: Home
    repo: config
    private: true
    note: "Configuration and content repo for this server"
  - name: Example Public Repo
    repo: my-public-repo
    private: false
    note: "A publicly-accessible repo"
  - name: Example Public Repo
    repo: my-private-repo
    private: true
    note: "A private repo"

# Authorized users. Admins have full access to all repos. Regular users
# can read all repos and push to their collab-repos.
users:
  - name: Beatrice
    admin: true
    public-key:
        KEY TEXT
  - name: Frankie
    collab-repos:
      - my-public-repo
      - my-private-repo
    public-key:
        KEY TEXT
```

When `soft` is run for the first time, it creates a configuration repo
containing the main README displayed in the TUI as well as a config file for
user access control.

```
git clone ssh://localhost:23231/config
```

The `config` repo is publicly writable by default, so be sure to setup your
access as desired. You can also set the `SOFT_SERVE_INITIAL_ADMIN_KEY`
environment variable before first run and it will restrict access to that
initial public key until you configure things otherwise.

## Pushing (and creating!) repos

You can add your Soft Serve server as a remote to any existing repo:

```
git remote add soft ssh://localhost:23231/REPO
```

After you’ve added the remote just go ahead and push. If the repo doesn’t exist
on the server it’ll be created.

```
git push soft main
```

## The Soft Serve TUI

Soft Serve serves a TUI over SSH for browsing repos, viewing READMEs, and
grabbing clone commands:

```
ssh localhost -p 23231
```

It's also possible to “link” to a specific repo:

```
ssh localhost -t -p 23231 REPO
```

### Server Settings

In addition to the Git-based configuration above, there are a few
environment-level settings:

* `SOFT_SERVE_PORT`: SSH listen port (_default 23231_)
* `SOFT_SERVE_HOST`: SSH listen host (_default 0.0.0.0_)
* `SOFT_SERVE_KEY_PATH`: SSH host key-pair path (_default .ssh/soft_serve_server_ed25519_)
* `SOFT_SERVE_REPO_PATH`: Path where repos are stored (_default .repos_)
* `SOFT_SERVE_INITIAL_ADMIN_KEY`: The public key that will initially have admin access to repos (_default ""_). This must be set before `soft` runs for the first time and creates the `config` repo. If set after the `config` repo has been created, this setting has no effect.
