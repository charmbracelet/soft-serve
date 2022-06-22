#!/bin/sh
set -e

MAIN=$1
PROJECT_NAME=$2

rm -rf completions
mkdir completions
for sh in bash zsh fish; do
	go run $MAIN completion "$sh" >"completions/$PROJECT_NAME.$sh"
done
