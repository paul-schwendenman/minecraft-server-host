#!/usr/bin/env bash
set -euxo pipefail

SRC_DIR="$(dirname "$0")/worlds"

sudo install -Dm755 "${SRC_DIR}/backup-worlds.sh" /usr/local/bin/backup-worlds.sh

sudo install -Dm644 "${SRC_DIR}/minecraft-world-backup@.service" /etc/systemd/system/minecraft-world-backup@.service
sudo install -Dm644 "${SRC_DIR}/minecraft-world-backup@.timer" /etc/systemd/system/minecraft-world-backup@.timer
sudo install -Dm644 "${SRC_DIR}/minecraft-world-backup.service" /etc/systemd/system/minecraft-world-backup.service
sudo install -Dm644 "${SRC_DIR}/minecraft-world-backup.timer" /etc/systemd/system/minecraft-world-backup.timer

sudo systemctl daemon-reexec
sudo systemctl daemon-reload
# leave timers disabled by default
