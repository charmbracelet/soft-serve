#!/bin/sh
set -e

MAIN=$1
PROJECT_NAME=$2

rm -rf manpages
mkdir manpages
go run $MAIN man | gzip -c >manpages/$PROJECT_NAME.1.gz
