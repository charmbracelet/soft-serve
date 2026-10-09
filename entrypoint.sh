#!/bin/sh
chown -R ${PUID:-1000}:${PGID:-1000} /soft-serve
exec su-exec "${PUID:-1000}:${PGID:-1000}" "$@"
