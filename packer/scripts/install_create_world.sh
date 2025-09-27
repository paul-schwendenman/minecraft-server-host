#!/usr/bin/env bash
set -euxo pipefail

# Drop the create-world helper
sudo tee /usr/local/bin/create-world.sh >/dev/null <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 2 ]]; then
  echo "Usage: $0 <world-name> <jar-version>"
  echo "Example: $0 vanilla-121 1.21.1"
  exit 1
fi

WORLD=$1
VERSION=$2
WORLD_DIR="/srv/minecraft-server/${WORLD}"
JAR_PATH="/opt/minecraft/jars/minecraft_server_${VERSION}.jar"
MAP_DIR="/var/www/map/${WORLD}"

if [[ ! -f "$JAR_PATH" ]]; then
  echo "Error: Jar for version $VERSION not found at $JAR_PATH"
  exit 1
fi

# --- World directory ---
sudo mkdir -p "$WORLD_DIR"
sudo chown -R minecraft:minecraft "$WORLD_DIR"

# EULA
if [[ ! -f "${WORLD_DIR}/eula.txt" ]]; then
  echo "eula=true" | sudo tee "${WORLD_DIR}/eula.txt" >/dev/null
  sudo chown minecraft:minecraft "${WORLD_DIR}/eula.txt"
fi

# Link jar
ln -s "$JAR_PATH" "${WORLD_DIR}/server.jar" || true

# server.properties (RCON + MOTD)
if [[ ! -f "${WORLD_DIR}/server.properties" ]]; then
  RCON_PASS=$(openssl rand -hex 12)
  cat <<EOPROP | sudo tee "${WORLD_DIR}/server.properties" >/dev/null
enable-rcon=true
rcon.port=25575
rcon.password=${RCON_PASS}
motd=Welcome to ${WORLD}
level-name=world
EOPROP
  sudo chown minecraft:minecraft "${WORLD_DIR}/server.properties"
fi

# --- Systemd enable ---
sudo systemctl enable "minecraft@${WORLD}"
sudo systemctl start "minecraft@${WORLD}"

# --- Map directory ---
sudo mkdir -p "$MAP_DIR"
sudo chown -R minecraft:minecraft "$MAP_DIR"

# --- Update Caddy config ---
CADDYFILE="/etc/caddy/Caddyfile"
BLOCK="
handle_path /map/${WORLD}/* {
    root * ${MAP_DIR}
    file_server
}
"

if ! grep -q "/map/${WORLD}" "$CADDYFILE"; then
  echo "$BLOCK" | sudo tee -a "$CADDYFILE" >/dev/null
  sudo systemctl reload caddy
fi

# --- Success ---
echo "World '${WORLD}' created with jar version ${VERSION}."
echo "Minecraft running as systemd unit minecraft@${WORLD}."
echo "Map served at http://<server>/map/${WORLD}/"
EOF

sudo chmod 0755 /usr/local/bin/create-world.sh
sudo chown root:root /usr/local/bin/create-world.sh
