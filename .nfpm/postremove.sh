#!/bin/sh
set -e

systemctl stop soft.service
systemctl disable soft.service
systemctl daemon-reload
