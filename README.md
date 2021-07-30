# Smoothie

Distribute your software on the command line with SSH and Git.

## What is it

Smoothie is a SSH server that hosts a Git server and interactive TUI built from
the repos you push to it. Authors can easily push their projects to Smoothie by
adding it as a remote and users can clone repos from Smoothie and stay up to
date with the TUI as you push commits.

## Pushing a repo

1. Run `smoothie`
2. Add smoothie as a remote on any git repo: `git remote add smoothie ssh://git@localhost:23231/smoothie`
3. Push stuff: `git push smoothie main`

## Cloning a repo

1. You'll need to know the name (for now, it's not listed anywhere): `git clone ssh://git@localhost:23231/smoothie`

## Smoothie TUI

If you `ssh localhost -p 23231` you'll see a list of the latest commits to the repos you've pushed.

## Auth

By default anyone can push or pull from the Git repositories. This is mainly
for testing, you can also whitelist public keys that have Git write access by
creating an authorized keys file with the public key for each author. By
default this file is expected to be at `./.ssh/smoothie_git_authorized_keys`.
