#!/usr/bin/env bash
set -euxo pipefail

SRC_DIR="/tmp/scripts/minecraft/maps"

# Install scripts
sudo install -Dm755 "${SRC_DIR}/rebuild-map.sh" /usr/local/bin/rebuild-map.sh
sudo install -Dm755 "${SRC_DIR}/build-map-manifests.sh" /usr/local/bin/build-map-manifests.sh

# Install service and timers
sudo install -Dm644 "${SRC_DIR}/minecraft-map-build@.service" /etc/systemd/system/minecraft-map-build@.service
sudo install -Dm644 "${SRC_DIR}/minecraft-map-build@.timer" /etc/systemd/system/minecraft-map-build@.timer
sudo install -Dm644 "${SRC_DIR}/minecraft-map-build-daily@.timer" /etc/systemd/system/minecraft-map-build-daily@.timer

sudo systemctl daemon-reexec
sudo systemctl daemon-reload
