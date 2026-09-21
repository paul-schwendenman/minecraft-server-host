#!/usr/bin/env bash
set -euxo pipefail

MINECRAFT_HOME="/srv/minecraft-server"
MAP_DIR="/var/www/map"
MINECRAFT_USER="minecraft"
MINECRAFT_GROUP="minecraft"

# --- Ensure user and group ---
# The numeric IDs are pinned because /srv/minecraft-server is a persistent EBS volume
# that outlives AMI rebuilds: file ownership on it is stored as numbers. With an
# unpinned `useradd -r` the IDs depend on what else was installed first, so an AMI
# rebuild can silently hand the volume to a different account (it once became caddy).
# If the ID is already taken the build fails loudly instead of drifting.
MINECRAFT_UID="${MINECRAFT_UID:-996}"
MINECRAFT_GID="${MINECRAFT_GID:-996}"

if ! getent group "${MINECRAFT_GROUP}" >/dev/null 2>&1; then
  sudo groupadd -r -g "${MINECRAFT_GID}" "${MINECRAFT_GROUP}"
fi

if ! id -u "${MINECRAFT_USER}" >/dev/null 2>&1; then
  sudo useradd -m -r -u "${MINECRAFT_UID}" -g "${MINECRAFT_GROUP}" -s /bin/bash "${MINECRAFT_USER}"
fi

if [[ "$(id -u "${MINECRAFT_USER}")" != "${MINECRAFT_UID}" || "$(getent group "${MINECRAFT_GROUP}" | cut -d: -f3)" != "${MINECRAFT_GID}" ]]; then
  echo "Error: ${MINECRAFT_USER} must be ${MINECRAFT_UID}:${MINECRAFT_GID} to match the persistent data volume" >&2
  exit 1
fi

sudo usermod -a -G "${MINECRAFT_GROUP}" "${MINECRAFT_USER}"
sudo mkdir -p "${MINECRAFT_HOME}"
sudo chown -R "${MINECRAFT_USER}:${MINECRAFT_GROUP}" "${MINECRAFT_HOME}"
sudo chmod g+s "${MINECRAFT_HOME}"

# -- Install nbt cli --
sudo -u minecraft pip3 install --user nbtlib

# --- Configure Caddy + map directory ---
sudo mkdir -p "${MAP_DIR}"
sudo chown -R root:${MINECRAFT_GROUP} "${MAP_DIR}"
sudo chmod -R 775 "${MAP_DIR}"