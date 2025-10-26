#!/usr/bin/env bash
set -euxo pipefail

MINECRAFT_GROUP="minecraft"
UNMINED_DIR="/opt/unmined"
MAP_DIR="/var/www/map"
UNMINED_VERSION="dev"

# -- Install nbt cli --
sudo -u minecraft pip3 install --user nbtlib

# --- Install uNmINeD CLI ---
if [[ ! -x "${UNMINED_DIR}/unmined-cli" ]]; then
  sudo rm -rf "${UNMINED_DIR}"
  sudo mkdir -p "${UNMINED_DIR}"

  TMPDIR=$(mktemp -d)
  wget -q -O "${TMPDIR}/unmined-cli.tgz" "https://unmined.net/download/unmined-cli-linux-x64-${UNMINED_VERSION}/"
  echo "9509f4aac9f8efac698931bd71f057bd3dbf8aaa717f6f158dc69ae0be33cfca  ${TMPDIR}/unmined-cli.tgz" | sha256sum -c -
  tar -xzf "${TMPDIR}/unmined-cli.tgz" -C "${TMPDIR}"

  EXTRACTED_DIR=$(find "${TMPDIR}" -maxdepth 1 -type d -name "unmined-cli_*_linux-x64" | head -n1)
  sudo mv "${EXTRACTED_DIR}"/* "${UNMINED_DIR}/"
  sudo chmod +x "${UNMINED_DIR}/unmined-cli"
  rm -rf "${TMPDIR}"
fi

# --- Install Caddy (official repo) ---
if ! command -v caddy >/dev/null 2>&1; then
  sudo apt-get install -y -qq debian-keyring debian-archive-keyring apt-transport-https
  curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' \
    | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
  curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' \
    | sudo tee /etc/apt/sources.list.d/caddy-stable.list >/dev/null
  sudo chmod o+r /usr/share/keyrings/caddy-stable-archive-keyring.gpg
  sudo chmod o+r /etc/apt/sources.list.d/caddy-stable.list
  sudo apt-get update -qq
  sudo apt-get install -y -qq caddy
fi

# --- Configure Caddy + map directory ---
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
