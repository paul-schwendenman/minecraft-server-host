#!/usr/bin/env bash
set -euxo pipefail

SRC_DIR="/tmp/scripts/minecraft"

# --- Install systemd units ---
sudo install -Dm644 "${SRC_DIR}/minecraft@.service" /etc/systemd/system/minecraft@.service

# Gate for the ExecStopPost drop-ins (world backup, map build) installed later
sudo install -Dm755 "${SRC_DIR}/minecraft-stop-gate.sh" /usr/local/bin/minecraft-stop-gate

sudo systemctl daemon-reexec
sudo systemctl daemon-reload
