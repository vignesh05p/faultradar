#!/usr/bin/env bash
set -euo pipefail

echo "Building FaultRadar..."
go build -o bin/faultradar ./cmd/faultradar

INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="${HOME}/.config/faultradar"

echo "Installing binary to ${INSTALL_DIR}..."
if [ -w "${INSTALL_DIR}" ]; then
  cp bin/faultradar "${INSTALL_DIR}/"
else
  echo "No write access to ${INSTALL_DIR}. Retrying with sudo..."
  sudo cp bin/faultradar "${INSTALL_DIR}/"
fi

echo "Copying default configuration..."
mkdir -p "${CONFIG_DIR}"
if [ ! -f "${CONFIG_DIR}/config.json" ]; then
  cp examples/config.json "${CONFIG_DIR}/config.json"
  echo "Created configuration at ${CONFIG_DIR}/config.json"
else
  echo "Configuration file already exists at ${CONFIG_DIR}/config.json, skipping overwrite."
fi

echo "FaultRadar installed successfully!"
