#!/usr/bin/env bash
set -euxo pipefail

SRC_DIR="/tmp/scripts/minecraft/user-data"
DEST_BIN="/usr/local/bin"

sudo install -Dm755 "${SRC_DIR}/mount-ebs.sh" "${DEST_BIN}/mount-ebs.sh"
sudo install -Dm755 "${SRC_DIR}/setup-env.sh" "${DEST_BIN}/setup-env.sh"
sudo install -Dm755 "${SRC_DIR}/setup-maps.sh" "${DEST_BIN}/setup-maps.sh"
sudo install -Dm755 "${SRC_DIR}/publish-dns.sh" "${DEST_BIN}/publish-dns.sh"
sudo install -Dm755 "${SRC_DIR}/configure-caddy.sh" "${DEST_BIN}/configure-caddy.sh"
