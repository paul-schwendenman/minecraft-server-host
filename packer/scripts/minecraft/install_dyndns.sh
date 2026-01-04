#!/usr/bin/env bash
set -euxo pipefail

SRC_DIR="/tmp/scripts/minecraft/dyndns"
DEST_ETC="/etc/systemd/system"

sudo install -Dm644 "${SRC_DIR}/dyndns.service" "${DEST_ETC}/dyndns.service"

sudo systemctl daemon-reload
sudo systemctl enable dyndns.service
