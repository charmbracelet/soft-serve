# Soft-Serve

Distribute your software on the command line with SSH and Git.

## What is it

Soft-Serve is a SSH server that hosts a Git server and interactive TUI built from
the repos you push to it. Authors can easily push their projects to Soft-Serve by
adding it as a remote and users can clone repos from Soft-Serve and stay up to
date with the TUI as you push commits.

## Pushing a repo

1. Run `soft-serve`
2. Add soft-serve as a remote on any git repo: `git remote add soft-serve ssh://git@localhost:23231/soft-serve`
3. Push stuff: `git push soft-serve main`

## Cloning a repo

1. You'll need to know the name (for now, it's not listed anywhere): `git clone ssh://git@localhost:23231/soft-serve`

## Soft-Serve TUI

If you `ssh localhost -p 23231` you'll see a list of the latest commits to the repos you've pushed.

## Auth

By default anyone can push or pull from the Git repositories. This is mainly
for testing, you can also whitelist public keys that have Git write access by
creating an authorized keys file with the public key for each author. By
default this file is expected to be at `./.ssh/soft_serve_git_authorized_keys`.
