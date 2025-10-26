#!/usr/bin/env bash
set -euxo pipefail

MINECRAFT_HOME="/srv/minecraft-server"
MINECRAFT_USER="minecraft"
MINECRAFT_GROUP="minecraft"

# --- Ensure user and group ---
if ! id -u "${MINECRAFT_USER}" >/dev/null 2>&1; then
  sudo useradd -m -r -s /bin/bash "${MINECRAFT_USER}"
fi

if ! getent group "${MINECRAFT_GROUP}" >/dev/null 2>&1; then
  sudo groupadd -r "${MINECRAFT_GROUP}"
fi

sudo usermod -a -G "${MINECRAFT_GROUP}" "${MINECRAFT_USER}"
sudo mkdir -p "${MINECRAFT_HOME}"
sudo chown -R "${MINECRAFT_USER}:${MINECRAFT_GROUP}" "${MINECRAFT_HOME}"
