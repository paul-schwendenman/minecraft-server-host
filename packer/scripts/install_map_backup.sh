#!/usr/bin/env bash
set -euxo pipefail

SRC_DIR="$(dirname "$0")/maps"

sudo install -Dm755 "${SRC_DIR}/backup-maps.sh" /usr/local/bin/backup-maps.sh
sudo install -Dm644 "${SRC_DIR}/minecraft-map-backup@.service" /etc/systemd/system/minecraft-map-backup@.service
sudo install -Dm644 "${SRC_DIR}/minecraft-map-backup@.timer" /etc/systemd/system/minecraft-map-backup@.timer

sudo systemctl daemon-reload

# No enable here — backup is manual or opt-in timer
# sudo systemctl enable --now minecraft-map-backup@world.timer
