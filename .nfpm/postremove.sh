#!/bin/sh
set -e

if ! command -V systemctl >/dev/null 2>&1; then
	echo "Not running SystemD, ignoring"
	exit 0
fi

systemctl stop soft.service
systemctl disable soft.service
systemctl daemon-reload
