#!/usr/bin/env bash
set -euxo pipefail

SRC_DIR="/tmp/scripts/minecraft/worlds"

# Backup services and timers
sudo install -Dm644 "${SRC_DIR}/minecraft-world-backup@.service" /etc/systemd/system/minecraft-world-backup@.service
sudo install -Dm644 "${SRC_DIR}/minecraft-world-backup@.timer" /etc/systemd/system/minecraft-world-backup@.timer
sudo install -Dm644 "${SRC_DIR}/minecraft-world-backup.service" /etc/systemd/system/minecraft-world-backup.service
sudo install -Dm644 "${SRC_DIR}/minecraft-world-backup.timer" /etc/systemd/system/minecraft-world-backup.timer

# Prune service and timer
sudo install -Dm644 "${SRC_DIR}/minecraft-world-prune.service" /etc/systemd/system/minecraft-world-prune.service
sudo install -Dm644 "${SRC_DIR}/minecraft-world-prune.timer" /etc/systemd/system/minecraft-world-prune.timer

# Hook into minecraft@.service (ExecStopPost to backup on stop)
sudo mkdir -p /etc/systemd/system/minecraft@.service.d
sudo install -Dm644 "${SRC_DIR}/minecraft-override-world-backup.conf" /etc/systemd/system/minecraft@.service.d/minecraft-world-backup.conf

sudo systemctl daemon-reexec
sudo systemctl daemon-reload
# leave timers disabled by default
