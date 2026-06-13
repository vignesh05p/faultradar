#!/usr/bin/env bash
set -euo pipefail

echo "=== Running gofmt ==="
if [ -n "$(gofmt -l .)" ]; then
  echo "Gofmt check failed! Please run 'gofmt -w .' on the codebase."
  gofmt -l .
  exit 1
fi

echo "=== Running Vet ==="
go vet ./...

echo "=== Running Tests ==="
go test -v -race -cover ./...

echo "=== Running Build ==="
go build ./...

echo "All verification checks passed successfully!"
