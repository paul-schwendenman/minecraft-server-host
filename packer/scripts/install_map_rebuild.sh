#!/usr/bin/env bash
set -euo pipefail

SRC_DIR="/tmp/scripts/maps"
DST_DIR="/srv/minecraft-server/scripts"

echo "=== Installing map refresh scripts ==="

sudo mkdir -p "$DST_DIR"
sudo cp "$SRC_DIR"/*.sh "$DST_DIR/"
sudo chmod +x "$DST_DIR"/*.sh

# Copy and enable systemd units
sudo cp "$SRC_DIR"/*.service "$SRC_DIR"/*.timer /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable minecraft-map-rebuild@world1.timer || true

echo "✅ Map refresh installed"
