#!/usr/bin/env bash
set -euxo pipefail

SRC_DIR="/tmp/scripts/minecraft/maps"

# Install scripts
sudo install -Dm755 "${SRC_DIR}/rebuild-map.sh" /usr/local/bin/rebuild-map.sh
sudo install -Dm755 "${SRC_DIR}/build-map-manifests.sh" /usr/local/bin/build-map-manifests.sh

# Install service and timers
sudo install -Dm644 "${SRC_DIR}/minecraft-map-update@.service" /etc/systemd/system/minecraft-map-update@.service
sudo install -Dm644 "${SRC_DIR}/minecraft-map-update@.timer" /etc/systemd/system/minecraft-map-update@.timer
sudo install -Dm644 "${SRC_DIR}/minecraft-map-update-daily@.timer" /etc/systemd/system/minecraft-map-update-daily@.timer

# Hook into minecraft@.service (ExecStopPost to rebuild on stop)
sudo mkdir -p /etc/systemd/system/minecraft@.service.d
sudo install -Dm644 "${SRC_DIR}/minecraft-override-rebuild.conf" /etc/systemd/system/minecraft@.service.d/minecraft-map-update.conf

sudo systemctl daemon-reexec
sudo systemctl daemon-reload
