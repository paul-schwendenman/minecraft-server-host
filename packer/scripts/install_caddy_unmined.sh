#!/usr/bin/env bash
set -euxo pipefail

MINECRAFT_GROUP="minecraft"
UNMINED_DIR="/opt/unmined"
MAP_DIR="/var/www/map"

# --- uNmINeD CLI (grab latest linux-x64 dev build) ---
if [[ ! -x "${UNMINED_DIR}/unmined-cli" ]]; then
  sudo rm -rf "${UNMINED_DIR}"
  sudo mkdir -p "${UNMINED_DIR}"

  wget -O /tmp/unmined-cli.tgz "https://unmined.net/download/unmined-cli-linux-x64-dev/"
  tar -xzf /tmp/unmined-cli.tgz -C /tmp

  EXTRACTED_DIR=$(find /tmp -maxdepth 1 -type d -name "unmined-cli_*_linux-x64" | head -n1)
  if [[ -z "$EXTRACTED_DIR" ]]; then
    echo "Could not find extracted unmined directory" >&2
    exit 1
  fi

  sudo mv "$EXTRACTED_DIR"/* "${UNMINED_DIR}/"
  sudo chmod +x "${UNMINED_DIR}/unmined-cli"
fi

# --- Caddy (official repo) ---
if ! command -v caddy >/dev/null 2>&1; then
  sudo apt-get install -y -qq debian-keyring debian-archive-keyring apt-transport-https
  curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' \
    | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
  curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' \
    | sudo tee /etc/apt/sources.list.d/caddy-stable.list
  sudo chmod o+r /usr/share/keyrings/caddy-stable-archive-keyring.gpg
  sudo chmod o+r /etc/apt/sources.list.d/caddy-stable.list
  sudo apt-get update -qq
  sudo apt-get install -y -qq caddy
fi

# Map dir + Caddyfile
sudo mkdir -p "${MAP_DIR}"
sudo chown -R root:${MINECRAFT_GROUP} "${MAP_DIR}"
sudo chmod -R 775 "${MAP_DIR}"

sudo tee /etc/caddy/Caddyfile >/dev/null <<EOF
:80 {
    root * ${MAP_DIR}
    file_server
}
EOF

sudo systemctl enable caddy
sudo systemctl restart caddy

# --- Map rebuild script (callable for any world path) ---
sudo tee /usr/local/bin/rebuild-map.sh >/dev/null <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

WORLD_PATH="${1:-/srv/minecraft-server/world}"

# If WORLD_PATH is a glob, expand it and recurse
if [[ "$WORLD_PATH" == *"*"* ]]; then
  for w in $WORLD_PATH; do
    [ -d "$w" ] && "$0" "$w"
  done
  exit 0
fi

MAP_ROOT="/var/www/map"
UNMINED="/opt/unmined/unmined-cli"

if [[ ! -x "${UNMINED}" ]]; then
  echo "Error: unmined-cli not installed at ${UNMINED}" >&2
  exit 1
fi

# Derive a world name from the path (basename of parent dir)
WORLD_NAME=$(basename "$(dirname "$WORLD_PATH")")
MAP_DIR="${MAP_ROOT}/${WORLD_NAME}"

echo "Rebuilding map for world '${WORLD_NAME}' -> ${MAP_DIR}"
mkdir -p "${MAP_DIR}"

"${UNMINED}" web render \
  --world="${WORLD_PATH}" \
  --output="${MAP_DIR}" \
  --zoomout=6 \
  --zoomin=4 \
  --shadows=3d \
  --players
EOF
sudo chmod 0755 /usr/local/bin/rebuild-map.sh

# Drop-in to hook map rebuild after stop for both units
sudo mkdir -p /etc/systemd/system/minecraft.service.d
sudo tee /etc/systemd/system/minecraft.service.d/override.conf >/dev/null <<'EOF'
[Service]
ExecStopPost=/usr/local/bin/rebuild-map.sh /srv/minecraft-server/world
EOF

sudo mkdir -p /etc/systemd/system/minecraft@.service.d
sudo tee /etc/systemd/system/minecraft@.service.d/override.conf >/dev/null <<'EOF'
[Service]
ExecStopPost=/usr/local/bin/rebuild-map.sh /srv/minecraft-server/%i/world
EOF

sudo systemctl daemon-reexec
sudo systemctl daemon-reload
