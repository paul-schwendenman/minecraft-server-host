#!/usr/bin/env bash
set -euxo pipefail

SRC_DIR="/tmp/scripts/minecraft/active-world"
DEST_BIN="/usr/local/bin"
DEST_ETC="/etc/systemd/system"

sudo install -Dm755 "${SRC_DIR}/minecraft-active.sh" "${DEST_BIN}/minecraft-active.sh"
sudo install -Dm644 "${SRC_DIR}/minecraft-active.service" "${DEST_ETC}/minecraft-active.service"

sudo systemctl daemon-reload
sudo systemctl enable minecraft-active.service
