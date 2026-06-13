#!/usr/bin/env bash
set -euo pipefail

INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="${HOME}/.config/faultradar"
PURGE=false

if [ "${1:-}" = "--purge" ]; then
  PURGE=true
fi

echo "Removing binary from ${INSTALL_DIR}..."
if [ -f "${INSTALL_DIR}/faultradar" ]; then
  if [ -w "${INSTALL_DIR}" ]; then
    rm "${INSTALL_DIR}/faultradar"
  else
    echo "No write access to ${INSTALL_DIR}. Retrying with sudo..."
    sudo rm "${INSTALL_DIR}/faultradar"
  fi
  echo "Removed binary."
else
  echo "Binary not found in ${INSTALL_DIR}."
fi

if [ "${PURGE}" = true ]; then
  echo "Purging configuration directory ${CONFIG_DIR}..."
  rm -rf "${CONFIG_DIR}"
  echo "Configuration purged."
else
  echo "Configuration directory kept at ${CONFIG_DIR}. Run with --purge to remove it."
fi

echo "FaultRadar uninstalled successfully!"
