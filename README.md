Soft Serve
==========

<p>
    <img style="width: 451px" src="https://stuff.charm.sh/soft-serve/soft-serve-header.png?0" alt="A nice rendering of some melting ice cream with the words ‘Charm Soft Serve’ next to it"><br>
    <a href="https://github.com/charmbracelet/soft-serve/releases"><img src="https://img.shields.io/github/release/charmbracelet/soft-serve.svg" alt="Latest Release"></a>
    <a href="https://pkg.go.dev/github.com/charmbracelet/soft-serve?tab=doc"><img src="https://godoc.org/github.com/golang/gddo?status.svg" alt="GoDoc"></a>
    <a href="https://github.com/charmbracelet/soft-serve/actions"><img src="https://github.com/charmbracelet/soft-serve/workflows/build/badge.svg" alt="Build Status"></a>
    <a href="https://nightly.link/charmbracelet/soft-serve/workflows/nightly/main"><img src="https://shields.io/badge/-Nightly%20Builds-orange?logo=hackthebox&logoColor=fff&style=appveyor"/></a>
</p>

A tasty, self-hostable Git server for the command line. 🍦

<picture>
  <source media="(max-width: 750px)" srcset="https://stuff.charm.sh/soft-serve/soft-serve-demo.gif?0">
  <source media="(min-width: 750px)" width="750" srcset="https://stuff.charm.sh/soft-serve/soft-serve-demo.gif?0">
  <img src="https://stuff.charm.sh/soft-serve/soft-serve-demo.gif?0" alt="Soft Serve screencast">
</picture>

* Configure with `git`
* Create repos on demand with `git push`
* Browse repos, files and commits with an SSH-accessible TUI
* TUI mouse support
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
ssh git.charm.sh ls soft-serve

# Print a specific file
ssh git.charm.sh cat soft-serve/cmd/soft/root.go

# Print a file with syntax highlighting and line numbers
ssh git.charm.sh cat soft-serve/cmd/soft/root.go -c -l
```

## Installation

Soft Serve is a single binary called `soft`. You can get it from a package
manager:

```bash
# macOS or Linux
brew tap charmbracelet/tap && brew install charmbracelet/tap/soft-serve

# Arch Linux
pacman -S soft-serve

# Nix
nix-env -iA nixpkgs.soft-serve

# Debian/Ubuntu
echo "deb https://repo.charm.sh/apt/ * *" | sudo tee /etc/apt/sources.list.d/charm.list
curl https://repo.charm.sh/apt/gpg.key | sudo apt-key add -
sudo apt update && sudo apt install soft-serve

# Fedora
echo '[charm]
name=Charm
baseurl=https://repo.charm.sh/yum/
enabled=1
gpgcheck=1
gpgkey=https://repo.charm.sh/yum/gpg.key' | sudo tee /etc/yum.repos.d/charm.repo
sudo yum install soft-serve
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

# Access level for anonymous users. Options are: admin-access, read-write,
# read-only, and no-access.
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

# Authorized users. Admins have full access to all repos. Private repos are only
# accessible by admins and collab users. Regular users can read public repos
# based on your anon-access setting.
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

When `soft serve` is run for the first time, it creates a configuration repo
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

<img src="https://stuff.charm.sh/soft-serve/soft-serve-demo-commit.png" width="750" alt="TUI example showing a diff">

Soft Serve serves a TUI over SSH for browsing repos, viewing files and commits,
and grabbing clone commands:

```
ssh localhost -p 23231
```

It's also possible to “link” to a specific repo:

```
ssh localhost -t -p 23231 REPO
```

You can copy text to your clipboard over SSH. For instance, you can press <kbd>c</kbd> on the highlighted repo in the menu to copy the clone command [^osc52].

[^osc52]: Copying over SSH depends on your terminal support of OSC52.

## The Soft Serve SSH CLI

```sh
$ ssh -p 23231 localhost help
Soft Serve is a self-hostable Git server for the command line.

Usage:
  ssh -p 23231 localhost [command]

Available Commands:
  cat         Outputs the contents of the file at path.
  git         Perform Git operations on a repository.
  help        Help about any command
  ls          List file or directory at path.
  reload      Reloads the configuration

Flags:
  -h, --help   help for ssh

Use "ssh -p 23231 localhost [command] --help" for more information about a command.
```

Soft Serve SSH CLI has the ability to print files and list directories, perform
`git` operations on remote repos, and reload the configuration when necessary.

To print a file tree for the project, just use the `list` command along with the
repo name as the SSH command to your Soft Serve server:

```sh
ssh -p 23231 localhost ls soft-serve
```

From there, you can print individual files using the `cat` command:

```sh
ssh -p 23231 localhost cat soft-serve/cmd/soft/root.go
```

You can add the `-c` flag to enable syntax coloring and `-l` to print line
numbers:

```sh
ssh -p 23231 localhost cat soft-serve/cmd/soft/root.go -c -l
```

You can also use the `git` command to perform Git operations on a repo such as changing the default branch name for instance:

```sh
ssh -p 23231 localhost git soft-serve symbolic-ref HEAD refs/heads/taco
```

Both `git` and `reload` commands need admin access to the server to work. So
make sure you have added your key as an admin user, or you’re using `anon-access:
admin-access` in the configuration.

## Managing Repos

`.repos` and `.ssh` directories are created when you first run `soft` at the paths specified for the `SOFT_SERVE_KEY_PATH` and `SOFT_SERVE_REPO_PATH` environment variables. 
It's recommended to have a dedicated directory for your soft-serve repos and config.

### Deleting a Repo

To delete a repo from your soft serve server, you'll have to remove the repo from the .repos directory.

### Renaming a Repo

To rename a repo's display name in the menu, change its name in the config.yaml file for your soft serve server.
By default, the display name will be the repository name. 

## A note about RSA keys

Unfortunately, due to a shortcoming in Go’s `x/crypto/ssh` package, Soft Serve
does not currently support access via new SSH RSA keys: only the old SHA-1
ones will work.

Until we sort this out you’ll either need an SHA-1 RSA key or a key with
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
