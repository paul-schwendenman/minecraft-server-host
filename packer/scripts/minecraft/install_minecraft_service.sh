#!/usr/bin/env bash
set -euxo pipefail

SRC_DIR="/tmp/scripts/minecraft"

# --- Install systemd units ---
sudo install -Dm644 "${SRC_DIR}/minecraft@.service" /etc/systemd/system/minecraft@.service

sudo systemctl daemon-reexec
sudo systemctl daemon-reload
