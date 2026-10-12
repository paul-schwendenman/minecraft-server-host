#!/usr/bin/env bash
set -euxo pipefail

SRC_DIR="/tmp/scripts/minecraft/maps"

# Install service and timers
sudo install -Dm644 "${SRC_DIR}/minecraft-map-build@.service" /etc/systemd/system/minecraft-map-build@.service
sudo install -Dm644 "${SRC_DIR}/minecraft-map-build@.timer" /etc/systemd/system/minecraft-map-build@.timer
sudo install -Dm644 "${SRC_DIR}/minecraft-map-build-daily@.timer" /etc/systemd/system/minecraft-map-build-daily@.timer

sudo systemctl daemon-reexec
sudo systemctl daemon-reload
