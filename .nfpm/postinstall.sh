#!/bin/sh
set -e

if ! command -V systemctl >/dev/null 2>&1; then
	echo "Not running SystemD, ignoring"
	exit 0
fi

systemd-sysusers
systemd-tmpfiles --create

systemctl daemon-reload
systemctl unmask soft-serve.service
systemctl preset soft-serve.service
