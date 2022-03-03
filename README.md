Soft Serve
==========

<p>
    <picture>
        <source srcset="https://stuff.charm.sh/soft-serve/soft-serve-header.webp?0" type="image/webp">
        <img style="width: 451px" src="https://stuff.charm.sh/soft-serve/soft-serve-header.png?0" alt="A nice rendering of some melting ice cream with the words ‘Charm Soft Serve’ next to it">
    </picture><br>
    <a href="https://github.com/charmbracelet/soft-serve/releases"><img src="https://img.shields.io/github/release/charmbracelet/soft-serve.svg" alt="Latest Release"></a>
    <a href="https://pkg.go.dev/github.com/charmbracelet/soft-serve?tab=doc"><img src="https://godoc.org/github.com/golang/gddo?status.svg" alt="GoDoc"></a>
    <a href="https://github.com/charmbracelet/soft-serve/actions"><img src="https://github.com/charmbracelet/soft-serve/workflows/build/badge.svg" alt="Build Status"></a>
    <a href="https://nightly.link/charmbracelet/soft-serve/workflows/nightly/main"><img src="https://shields.io/badge/-Nightly%20Builds-orange?logo=hackthebox&logoColor=fff&style=appveyor"/></a>
</p>

A tasty, self-hostable Git server for the command line. 🍦

<img src="https://stuff.charm.sh/soft-serve/soft-serve-cli-demo.gif?0" width="750" alt="Soft Serve screencast">

* Configure with `git`
* Create repos on demand with `git push`
* Browse repos, files and commits with an SSH-accessible TUI
* Print files over SSH with or without syntax highlighting and line numbers
* Easy access control
  - Allow/disallow anonymous access
  - Add collaborators with SSH public keys
  - Repos can be public or private

## Where can I see it?

Just run `ssh git.charm.sh` for an example. You can also try some of the following commands:

```bash
# Jump directly to a repo in the TUI
ssh git.charm.sh -t soft-serve

# Print out a directory tree for a repo
ssh git.charm.sh soft-serve

# Print a specific file
ssh git.charm.sh soft-serve/cmd/soft/main.go

# Print a file with syntax highlighting and line numbers
ssh git.charm.sh soft-serve/cmd/soft/main.go -c -l
```

## Installation

Soft Serve is a single binary called `soft`. You can get it from a package
manager:

```bash
# macOS or Linux
brew tap charmbracelet/tap && brew install charmbracelet/tap/soft-serve

# Arch Linux
pacman -S soft-serve
```

You can also download a binary from the [releases][releases] page. Packages are
available in Alpine, Debian, and RPM formats. Binaries are available for Linux,
macOS, and Windows.

[releases]: https://github.com/charmbracelet/soft-serve/releases

Or just install it with `go`:

```bash
go install github.com/charmbracelet/soft-serve/cmd/soft@latest
```

## Setting up a server

Make sure `git` is installed, then run `soft`. That’s it.

A [Docker image][docker] is also available.

[docker]: https://github.com/charmbracelet/soft-serve/blob/main/docker.md

## Configuration

The Soft Serve configuration is simple and straightforward:

```yaml
# The name of the server to show in the TUI.
name: Soft Serve

# The host and port to display in the TUI. You may want to change this if your
# server is accessible from a different host and/or port that what it's
# actually listening on (for example, if it's behind a reverse proxy).
host: localhost
port: 23231

# The access level for anonymous users. Options are: read-write, read-only
# and no-access.
anon-access: read-write

# You can grant read-only access to users without private keys.
allow-keyless: false

# Customize repos in the menu
repos:
  - name: Home
    repo: config
    private: true
    note: "Configuration and content repo for this server"
  - name: Example Public Repo
    repo: my-public-repo
    private: false
    note: "A publicly-accessible repo"
    readme: docs/README.md
  - name: Example Private Repo
    repo: my-private-repo
    private: true
    note: "A private repo"

# Authorized users. Admins have full access to all repos. Regular users
# can read all repos and push to their collab-repos.
users:
  - name: Beatrice
    admin: true
    public-keys:
      - ssh-rsa AAAAB3Nz...   # redacted
      - ssh-ed25519 AAAA...   # redacted
  - name: Frankie
    collab-repos:
      - my-public-repo
      - my-private-repo
    public-keys:
      - ssh-rsa AAAAB3Nz...   # redacted
      - ssh-ed25519 AAAA...   # redacted
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
If you're having trouble, make sure you have generated keys with `ssh-keygen`
as configuration is not supported for keyless users.

### Server Settings

In addition to the Git-based configuration above, there are a few
environment-level settings:

* `SOFT_SERVE_PORT`: SSH listen port (_default 23231_)
* `SOFT_SERVE_HOST`: Address to use in public clone URLs
* `SOFT_SERVE_BIND_ADDRESS`: Network interface to listen on (_default 0.0.0.0_)
* `SOFT_SERVE_KEY_PATH`: SSH host key-pair path (_default .ssh/soft_serve_server_ed25519_)
* `SOFT_SERVE_REPO_PATH`: Path where repos are stored (_default .repos_)
* `SOFT_SERVE_INITIAL_ADMIN_KEY`: The public key that will initially have admin access to repos (_default ""_). This must be set before `soft` runs for the first time and creates the `config` repo. If set after the `config` repo has been created, this setting has no effect.

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

<img src="https://stuff.charm.sh/soft-serve/soft-serve-tui-diff.png" width="750" alt="TUI example showing a diff">

Soft Serve serves a TUI over SSH for browsing repos, viewing files and commits,
and grabbing clone commands:

```
ssh localhost -p 23231
```

It's also possible to “link” to a specific repo:

```
ssh localhost -t -p 23231 REPO
```

You can use the `tab` key to move between the repo menu and a particular repo.
When a repo is highlighted you can use the following keys for navigation:

* `R` – View the project's README file
* `F` – Navigate and view all of the files in the project
* `C` – View the commit log for the repo
* `B` – View branches and tags for the repo

## Printing files over SSH

<img src="https://stuff.charm.sh/soft-serve/soft-serve-cat-example.png?1" width="750" alt="Example tree and file output on the CLI">

Soft Serve also has the ability to print files and directories directly from a
remote repo without the need to clone or use the TUI.

To print a file tree for the project, just use the repo name as the SSH command
to your Soft Serve server:

```
ssh git.charm.sh soft-serve
```

From there you can print individual files:

```
ssh git.charm.sh soft-serve/cmd/soft/main.go
```

You can add the `-c` flag to enable syntax coloring and `-l` to print line
numbers:

```
ssh git.charm.sh soft-serve/cmd/soft/main.go -c -l
```

### A note about RSA keys

Unfortunately, due to a shortcoming in Go’s `x/crypto/ssh` package, Soft Serve
does not currently support access via new SSH RSA keys: only the old SHA-1
ones will work.

Until we sort this out you'll either need an SHA-1 RSA key or a key with
another algorithm, e.g. Ed25519. Not sure what type of keys you have?
You can check with the following:

```
$ find ~/.ssh/id_*.pub -exec ssh-keygen -l -f {} \;
```

If you’re curious about the inner workings of this problem have a look at:

- https://github.com/golang/go/issues/37278
- https://go-review.googlesource.com/c/crypto/+/220037
- https://github.com/golang/crypto/pull/197

## Feedback

We’d love to hear your thoughts on this project. Feel free to drop us a note!

* [Twitter](https://twitter.com/charmcli)
* [The Fediverse](https://mastodon.technology/@charm)
* [Slack](https://charm.sh/slack)

## License

[MIT](https://github.com/charmbracelet/soft-serve/raw/main/LICENSE)

***

Part of [Charm](https://charm.sh).

<a href="https://charm.sh/"><img alt="The Charm logo" src="https://stuff.charm.sh/charm-badge.jpg" width="400"></a>

Charm热爱开源 • Charm loves open source
