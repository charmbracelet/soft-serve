#!/bin/sh
set -e

if ! command -V systemctl >/dev/null 2>&1; then
    echo "Not running SystemD, ignoring"
	exit 0
fi

echo "Enabling and starting soft.service"
systemctl daemon-reload
systemctl unmask soft.service
systemctl preset soft.service
systemctl enable soft.service
systemctl restart soft.service
