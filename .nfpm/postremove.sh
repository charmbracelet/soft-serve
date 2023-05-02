#!/bin/sh
set -e

if ! command -V systemctl >/dev/null 2>&1; then
	echo "Not running SystemD, ignoring"
	exit 0
fi

echo "Disabling and starting soft-server.service"
systemctl stop soft-serve.service
systemctl disable soft-serve.service
systemctl daemon-reload
systemctl reset-failed

echo "WARN: the soft-serve user/group and /var/lib/soft-serve directory were not removed"
