#!/usr/bin/env bash
set -euxo pipefail

SRC_DIR="/tmp/scripts/maps"

# Core scripts
sudo install -Dm755 "${SRC_DIR}/backup-maps.sh" /usr/local/bin/backup-maps.sh

# Per-world units
sudo install -Dm644 "${SRC_DIR}/minecraft-map-backup@.service" /etc/systemd/system/minecraft-map-backup@.service
sudo install -Dm644 "${SRC_DIR}/minecraft-map-backup@.timer" /etc/systemd/system/minecraft-map-backup@.timer

# Global units
sudo install -Dm644 "${SRC_DIR}/minecraft-map-backup.service" /etc/systemd/system/minecraft-map-backup.service
sudo install -Dm644 "${SRC_DIR}/minecraft-map-backup.timer" /etc/systemd/system/minecraft-map-backup.timer

sudo mkdir -p /etc/systemd/system/minecraft@.service.d
sudo install -Dm644 "${SRC_DIR}/minecraft-override-backup.conf" /etc/systemd/system/minecraft@.service.d/minecraft-backup.conf

sudo systemctl daemon-reexec
sudo systemctl daemon-reload

# No timers enabled by default
# sudo systemctl enable --now minecraft-map-backup@default.timer
# sudo systemctl enable --now minecraft-map-backup.timer
