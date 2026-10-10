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

# Shared restic cache. minecraftctl uses it when run as its owner, since the
# default (~/.cache) is hidden from minecraft@.service by ProtectHome.
sudo install -d -o minecraft -g minecraft -m 0750 /var/cache/restic

# Hook into minecraft@.service (ExecStopPost to backup on stop)
sudo mkdir -p /etc/systemd/system/minecraft@.service.d
sudo install -Dm644 "${SRC_DIR}/minecraft-override-world-backup.conf" /etc/systemd/system/minecraft@.service.d/minecraft-world-backup.conf

sudo systemctl daemon-reexec
sudo systemctl daemon-reload

# Weekly backup of the whole worlds dir (configs and other files no per-world
# backup covers; caddy/ excluded) and weekly restic prune, without which the
# repo grows forever. Enabled here because /etc comes from the AMI; per-world
# timers are enabled by `minecraftctl world create/register`.
sudo systemctl enable minecraft-world-backup.timer minecraft-world-prune.timer
