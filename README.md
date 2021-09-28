# Soft Serve

A tasty Git server. Self-hosted with a built in SSH powered TUI.

## What is it?

Soft Serve is a Git server that runs its own SSH service, allows repo creation
on first push, is configured by cloning a `config` repo and provides a TUI
accessible to anyone over SSH without having to worry about setting up accounts
on the host machine. Give it a shot!

```
ssh beta.charm.sh
```

## Installing / Building

The Soft Serve command is called `soft`. You can build it with `go`.

```
cd cmd/soft
go build
```

## Setting up a server

Make sure `git` is installed, then run `soft`.

## Configuring

When Soft Serve is run for the first time, it creates a configuration repo that
contains the README displayed for Home and user access control. By default the
`config` repo is publicly writable, so be sure to setup your access as desired.
You can also set the `SOFT_SERVE_AUTH_KEY` environment variable and it will
restrict access to that initial public key.

```
git clone ssh://localhost:23231/config
```

## Pushing a repo

You can add your Soft Serve server as a remote to any existing repo.

```
git remote add soft ssh://localhost:23231/REPO
```

After you've added the remote, you can push. If it's a new repo, it will be
automatically added to the server.

```
git push soft main
```

## Soft Serve TUI

Soft Serve provides a TUI over SSH to browse repos, view READMEs, and grab
clone commands.

```
ssh localhost -p 23231
```

It's also possible to direct link to a specific repo.

```
ssh localhost -t -p 23231 REPO
```

### Server Options

You have control over the various options via the following server environment
variables:

* `SOFT_SERVE_PORT` - SSH listen port (_default 23231_)
* `SOFT_SERVE_HOST` - SSH listen host (_default 0.0.0.0_)
* `SOFT_SERVE_KEY_PATH` - SSH host key-pair path (_default .ssh/soft_serve_server_ed25519_)
* `SOFT_SERVE_REPO_PATH` - Path where repos are stored (_default .repos_)
* `SOFT_SERVE_AUTH_KEY` - Initial admin public key (_default ""_)
