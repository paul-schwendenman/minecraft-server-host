#!/usr/bin/env bash

set -o nounset
set -o errexit
set -o pipefail
set -o verbose

#exec > >(tee /var/log/user-data.log|logger -t user-data -s 2>/dev/console) 2>&1

MINECRAFT_HOME="/srv/minecraft-server"
MINECRAFT_USER="minecraft"
MINECRAFT_GROUP="minecraft"
UNMINED_DIR="/opt/unmined"
MAP_DIR="/var/www/map"

# --- Install dependencies ---
sudo apt update -qq
sudo apt install -y unzip curl gnupg2

# --- Install uNmINeD CLI ---
if [[ ! -x "${UNMINED_DIR}/unmined-cli" ]]; then
    sudo rm -rf "${UNMINED_DIR}"
    sudo mkdir -p "${UNMINED_DIR}"

    wget -O /tmp/unmined-cli.tgz "https://unmined.net/download/unmined-cli-linux-x64-dev/"
    tar -xzf /tmp/unmined-cli.tgz -C /tmp

    # Move the extracted contents (versioned dir) into /opt/unmined
    EXTRACTED_DIR=$(find /tmp -maxdepth 1 -type d -name "unmined-cli_*_linux-x64" | head -n1)
    if [[ -z "$EXTRACTED_DIR" ]]; then
        echo "Could not find extracted unmined directory" >&2
        exit 1
    fi

    sudo mv "$EXTRACTED_DIR"/* "${UNMINED_DIR}/"
    sudo chmod +x "${UNMINED_DIR}/unmined-cli"
fi


# --- Install Caddy ---
if ! command -v caddy >/dev/null 2>&1; then
    sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https
    curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' \
      | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
    curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' \
      | sudo tee /etc/apt/sources.list.d/caddy-stable.list
    sudo chmod o+r /usr/share/keyrings/caddy-stable-archive-keyring.gpg
    sudo chmod o+r /etc/apt/sources.list.d/caddy-stable.list
    sudo apt update -qq
    sudo apt install -y caddy
fi

# Configure Caddy to serve map dir
sudo mkdir -p "${MAP_DIR}"
sudo tee /etc/caddy/Caddyfile > /dev/null << EOF
:80 {
    root * ${MAP_DIR}
    file_server
}
EOF
sudo systemctl enable caddy
sudo systemctl restart caddy

# --- Rebuild script ---
sudo tee /srv/minecraft-server/rebuild-map.sh > /dev/null << 'EOF'
#!/usr/bin/env bash
set -euo pipefail

WORLD_PATH="${1:-/srv/minecraft-server/world}"
MAP_DIR="/var/www/map"
UNMINED="/opt/unmined/unmined-cli"

if [[ ! -x "${UNMINED}" ]]; then
    echo "Error: unmined-cli not installed at ${UNMINED}" >&2
    exit 1
fi

echo "Rebuilding map from world: ${WORLD_PATH}"
mkdir -p "${MAP_DIR}"
"${UNMINED}" web render \
  --world="${WORLD_PATH}" \
  --output="${MAP_DIR}" \
  --zoomout=6 \
  --zoomin=4 \
  --shadows=3d \
  --players

echo "Map rebuild complete. Output at ${MAP_DIR}"
EOF

sudo chmod +x /srv/minecraft-server/rebuild-map.sh

# --- Systemd override for Minecraft service ---
sudo mkdir -p /etc/systemd/system/minecraft.service.d
sudo tee /etc/systemd/system/minecraft.service.d/override.conf > /dev/null << EOF
[Service]
ExecStopPost=/srv/minecraft-server/rebuild-map.sh
EOF

# Reload systemd to apply override
sudo systemctl daemon-reexec
sudo systemctl daemon-reload
