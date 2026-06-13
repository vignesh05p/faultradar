#!/usr/bin/env bash
set -euo pipefail

echo "Building release binaries..."
mkdir -p bin

echo "Building linux/amd64..."
GOOS=linux GOARCH=amd64 go build -o bin/faultradar-linux-amd64 ./cmd/faultradar

echo "Building linux/arm64..."
GOOS=linux GOARCH=arm64 go build -o bin/faultradar-linux-arm64 ./cmd/faultradar

echo "Release binaries built in bin/:"
ls -lh bin/
