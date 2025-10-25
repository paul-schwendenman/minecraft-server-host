#!/usr/bin/env bash
set -euxo pipefail

SRC_DIR="$(dirname "$0")/maps"

# Core scripts
sudo install -Dm755 "${SRC_DIR}/backup-maps.sh" /usr/local/bin/backup-maps.sh

# Per-world units
sudo install -Dm644 "${SRC_DIR}/minecraft-map-backup@.service" /etc/systemd/system/minecraft-map-backup@.service
sudo install -Dm644 "${SRC_DIR}/minecraft-map-backup@.timer" /etc/systemd/system/minecraft-map-backup@.timer

# Global units
sudo install -Dm644 "${SRC_DIR}/minecraft-map-backup.service" /etc/systemd/system/minecraft-map-backup.service
sudo install -Dm644 "${SRC_DIR}/minecraft-map-backup.timer" /etc/systemd/system/minecraft-map-backup.timer

# Add to minecraft@.service.d override if present
if [[ -d /etc/systemd/system/minecraft@.service.d ]]; then
  sudo install -Dm644 "${SRC_DIR}/minecraft-override.conf" /etc/systemd/system/minecraft@.service.d/map-backup.conf
fi

sudo systemctl daemon-reexec
sudo systemctl daemon-reload

# No timers enabled by default
# sudo systemctl enable --now minecraft-map-backup@default.timer
# sudo systemctl enable --now minecraft-map-backup.timer
