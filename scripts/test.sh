#!/usr/bin/env bash
set -euo pipefail

echo "=== Running Vet ==="
go vet ./...

echo "=== Running Tests ==="
go test -v -race -cover ./...

echo "=== Running Build ==="
mkdir -p bin
go build -o bin/faultradar ./cmd/faultradar

echo "All tests passed successfully!"
