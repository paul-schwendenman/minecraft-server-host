#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 2 ]]; then
  echo "Usage: $0 <world-name> <jar-version> [seed]"
  echo "Example: $0 vanilla-121 1.21.1 8675309"
  exit 1
fi

WORLD="$1"
VERSION="$2"
SEED="${3:-}"

WORLD_DIR="/srv/minecraft-server/${WORLD}"
JAR_PATH="/opt/minecraft/jars/minecraft_server_${VERSION}.jar"
MAP_DIR="/var/www/map/${WORLD}"
CONFIG_FILE="${WORLD_DIR}/map-config.yml"

if [[ ! -f "$JAR_PATH" ]]; then
  echo "Error: Minecraft server jar for version $VERSION not found at $JAR_PATH" >&2
  exit 1
fi

# --- World directory ---
sudo mkdir -p "$WORLD_DIR"
sudo chown -R minecraft:minecraft "$WORLD_DIR"

# Link the server jar
sudo -u minecraft ln -sf "$JAR_PATH" "${WORLD_DIR}/server.jar"

# Accept EULA
sudo -u minecraft tee "${WORLD_DIR}/eula.txt" >/dev/null <<'EOEULA'
eula=true
EOEULA

# Load shared RCON settings
if [[ -r /etc/minecraft.env ]]; then
  source /etc/minecraft.env
else
  echo "Warning: /etc/minecraft.env not found, skipping RCON configuration"
fi

# server.properties (RCON + MOTD)
if [[ ! -f "${WORLD_DIR}/server.properties" ]]; then
  sudo -u minecraft tee "${WORLD_DIR}/server.properties" >/dev/null <<EOF
enable-rcon=true
rcon.port=${RCON_PORT:-25575}
rcon.password=${RCON_PASSWORD}
motd=Welcome to ${WORLD}
level-name=world
EOF
  if [[ -n "$SEED" ]]; then
    echo "level-seed=${SEED}" | sudo tee -a "${WORLD_DIR}/server.properties" >/dev/null
  fi
fi

# --- Systemd enable ---
sudo systemctl enable "minecraft@${WORLD}.service"
sudo systemctl start "minecraft@${WORLD}.service"
#sudo systemctl enable --now "minecraft@${WORLD}.service"

# --- Map config ---

if [[ ! -f "$CONFIG_FILE" ]]; then
  echo "→ Creating default map-config.yml for ${WORLD}"

  cat >"$CONFIG_FILE" <<'YAML'
# Default map configuration for uNmINeD
# Adjust zoom levels, dimensions, and regions as needed

defaults:
  zoomout: 2
  zoomin: 1
  imageformat: jpeg
  chunkprocessors: 4

maps:
  - name: overworld
    dimension: overworld
    output_subdir: overworld
    options:
      shadows: 3d
    ranges:
      - name: spawn_area
        center: [0, 0]
        radius: 2048
        zoomout: 2
        zoomin: 4

  - name: nether
    dimension: nether
    output_subdir: nether
    options:
      topY: 68
      shadows: 2d
      night: false
    ranges:
      - name: hub
        center: [0, 0]
        radius: 1024
        zoomout: 3
        zoomin: 2

  - name: end
    dimension: end
    output_subdir: end
    options:
      shadows: 2d
YAML

  chown minecraft:minecraft "$CONFIG_FILE"
  chmod 644 "$CONFIG_FILE"
else
  echo "→ map-config.yml already exists, skipping creation"
fi

# --- Map directory ---
sudo mkdir -p "$MAP_DIR"
sudo chown -R minecraft:minecraft "$MAP_DIR"

# --- Success ---
echo "World '${WORLD}' created with jar version ${VERSION}."
echo "Minecraft running as systemd unit minecraft@${WORLD}."
echo "Map served at http://<server>/map/${WORLD}/"
