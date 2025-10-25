#!/usr/bin/env bash
set -euxo pipefail

SRC_DIR="$(dirname "$0")/autoshutdown"
DEST_BIN="/usr/local/bin"
DEST_ETC="/etc/systemd/system"
DEST_SUDOERS="/etc/sudoers.d"

sudo install -Dm755 "${SRC_DIR}/autoshutdown.sh" "${DEST_BIN}/autoshutdown.sh"
sudo install -Dm644 "${SRC_DIR}/autoshutdown.service" "${DEST_ETC}/autoshutdown.service"
sudo install -Dm644 "${SRC_DIR}/autoshutdown.timer" "${DEST_ETC}/autoshutdown.timer"
sudo install -Dm440 "${SRC_DIR}/minecraft-shutdown.sudoers" "${DEST_SUDOERS}/minecraft-shutdown"

sudo systemctl daemon-reload
sudo systemctl enable autoshutdown.timer
