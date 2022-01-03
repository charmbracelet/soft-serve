# Running Soft-Serve with Docker

The official Soft Serve Docker images are available at [charmcli/soft-serve][docker]. Development and nightly builds are available at [ghcr.io/charmbracelet/soft-serve][ghcr]

```sh
docker pull charmcli/soft-serve:latest
```

Here’s how you might run `soft-serve` as a container.  Keep in mind that
repositories are stored in the `/soft-serve` directory, so you’ll likely want
to mount that directory as a volume in order keep your repositories backed up.

```sh
docker run \
  --name=soft-seve \
  --volume /path/to/data:/soft-serve \
  --publish 23231:23231 \
  --restart unless-stopped \
  charmcli/soft-serve:latest
```

Or by using docker-compose:

```yaml
---
version: "3.1"
services:
  soft-serve:
    image: charmcli/soft-serve:latest
    container_name: soft-serve
    volumes:
      - /path/to/data:/soft-serve
    ports:
      - 23231:23231
    restart: unless-stopped
```

[docker]: https://hub.docker.com/r/charmcli/soft-serve
[ghcr]: https://github.com/charmbracelet/soft-serve/pkgs/container/soft-serve

***

Part of [Charm](https://charm.sh).

<a href="https://charm.sh/"><img alt="The Charm logo" src="https://stuff.charm.sh/charm-badge-unrounded.jpg" width="400"></a>

Charm热爱开源 • Charm loves open source
