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
  --name=soft-serve \
  --volume /path/to/data:/soft-serve \
  --publish 23231:23231 \
  --publish 23232:23232 \
  --publish 23233:23233 \
  --publish 9418:9418 \
  -e SOFT_SERVE_INITIAL_ADMIN_KEYS="YOUR_ADMIN_KEY_HERE" \
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
      - 23232:23232
      - 23233:23233
      - 9418:9418
    environment:
      SOFT_SERVE_INITIAL_ADMIN_KEYS: "YOUR_ADMIN_KEY_HERE"
    restart: unless-stopped
```

[docker]: https://hub.docker.com/r/charmcli/soft-serve
[ghcr]: https://github.com/charmbracelet/soft-serve/pkgs/container/soft-serve


> **Warning**
>
> Make sure to run the image without a TTY, i.e.: do not use the `--tty`/`-t`
> flags.


***

Part of [Charm](https://charm.sh).

<a href="https://charm.sh/"><img alt="The Charm logo" src="https://stuff.charm.sh/charm-badge-unrounded.jpg" width="400"></a>

Charm热爱开源 • Charm loves open source
