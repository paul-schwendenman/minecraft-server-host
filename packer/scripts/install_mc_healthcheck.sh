#!/usr/bin/env bash
set -euxo pipefail

SRC_DIR="/tmp/scripts/health"

sudo install -Dm755 "${SRC_DIR}/mc-healthcheck.sh" /usr/local/bin/mc-healthcheck.sh
sudo install -Dm644 "${SRC_DIR}/minecraft-health.service" /etc/systemd/system/minecraft-health.service

sudo systemctl daemon-reexec
sudo systemctl daemon-reload
