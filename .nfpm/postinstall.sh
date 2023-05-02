#!/bin/sh
set -e

if ! command -V systemctl >/dev/null 2>&1; then
	echo "Not running SystemD, ignoring"
	exit 0
fi

systemd-sysusers
systemd-tmpfiles --create

echo "Enabling and starting soft-server.service"
systemctl daemon-reload
systemctl unmask soft-serve.service
systemctl preset soft-serve.service
systemctl enable soft-serve.service
systemctl restart soft-serve.service
