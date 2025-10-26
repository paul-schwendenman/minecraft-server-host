#!/usr/bin/env bash
set -euxo pipefail

SRC_DIR="/tmp/scripts/maps"

sudo install -Dm755 "${SRC_DIR}/rebuild-map.sh" /usr/local/bin/rebuild-map.sh
sudo install -Dm755 "${SRC_DIR}/build-map-manifests.sh" /usr/local/bin/build-map-manifests.sh

sudo install -Dm644 "${SRC_DIR}/minecraft-map-rebuild@.service" /etc/systemd/system/minecraft-map-rebuild@.service
sudo install -Dm644 "${SRC_DIR}/minecraft-map-rebuild@.timer" /etc/systemd/system/minecraft-map-rebuild@.timer

# Hook into minecraft@.service (ExecStopPost)
sudo mkdir -p /etc/systemd/system/minecraft@.service.d
sudo install -Dm644 "${SRC_DIR}/minecraft-override.conf" /etc/systemd/system/minecraft@.service.d/override.conf

sudo systemctl daemon-reexec
sudo systemctl daemon-reload
sudo systemctl enable minecraft-map-rebuild@world.timer || true
