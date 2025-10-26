#!/usr/bin/env bash
set -euxo pipefail

SRC_DIR="/tmp/scripts/create-world"
DEST_BIN="/usr/local/bin"

sudo install -Dm755 "${SRC_DIR}/create-world.sh" "${DEST_BIN}/create-world.sh"
