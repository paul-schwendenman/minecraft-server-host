#!/usr/bin/env bash
set -euxo pipefail

SRC_DIR="$(dirname "$0")/create-world"
DEST_BIN="/usr/local/bin"

sudo install -Dm755 "${SRC_DIR}/create-world.sh" "${DEST_BIN}/create-world.sh"
