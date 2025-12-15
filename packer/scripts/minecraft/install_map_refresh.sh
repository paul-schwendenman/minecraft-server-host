#!/usr/bin/env bash
set -euxo pipefail

SRC_DIR="/tmp/scripts/minecraft/maps"

sudo install -Dm644 "${SRC_DIR}/minecraft-map-refresh@.service" /etc/systemd/system/minecraft-map-refresh@.service
sudo install -Dm644 "${SRC_DIR}/minecraft-map-refresh@.timer" /etc/systemd/system/minecraft-map-refresh@.timer

sudo systemctl daemon-reexec
sudo systemctl daemon-reload
