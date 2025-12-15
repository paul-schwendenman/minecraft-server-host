#!/usr/bin/env bash
set -euxo pipefail

# --- Update system packages (base AMI might be old) ---
sudo apt-get update -qq
sudo DEBIAN_FRONTEND=noninteractive apt-get upgrade -y -qq

# --- Base OS packages ---
sudo add-apt-repository -y universe
sudo apt-get update -qq
sudo apt-get install -y -qq \
  openjdk-21-jre-headless \
  screen unzip wget curl ca-certificates \
  python3-pip git build-essential jq xfsprogs

# --- Python tools (optional) ---
pip3 install --user --upgrade mcstatus nbtlib

# --- AWS CLI v2 ---
if ! command -v aws >/dev/null 2>&1; then
  echo "[*] Installing AWS CLI v2 with GPG verification"

  TMPDIR=$(mktemp -d)
  cd "$TMPDIR"

  # Fetch the CLI package and signature
  curl -s -O https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip
  curl -s -O https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip.sig
  gpg --import /tmp/scripts/base/deps/awscliv2.pub

  # Verify the signature
  echo "[*] Verifying AWS CLI package signature..."
  gpg --verify awscli-exe-linux-x86_64.zip.sig awscli-exe-linux-x86_64.zip

  # Install only if verification passes
  unzip -q awscli-exe-linux-x86_64.zip -d .
  sudo ./aws/install

  cd /
  rm -rf "$TMPDIR"
fi

# --- yq (Go binary, verified checksum) ---
if ! command -v yq >/dev/null 2>&1; then
  echo "[*] Installing yq (Go binary, verified checksum)"
  YQ_VERSION="v4.48.1"
  cd /tmp
  wget -q "https://github.com/mikefarah/yq/releases/download/${YQ_VERSION}/yq_linux_amd64" -O yq_linux_amd64
  echo "99df6047f5b577a9d25f969f7c3823ada3488de2e2115b30a0abb10d9324fd9f  yq_linux_amd64" | sha256sum -c -

  sudo mv yq_linux_amd64 /usr/local/bin/yq
  sudo chmod 0755 /usr/local/bin/yq
  sudo chown root:root /usr/local/bin/yq
fi

# --- Install uNmINeD CLI ---
MINECRAFT_GROUP="minecraft"
UNMINED_DIR="/opt/unmined"
MAP_DIR="/var/www/map"
UNMINED_VERSION="dev"

if [[ ! -x "${UNMINED_DIR}/unmined-cli" ]]; then
  sudo rm -rf "${UNMINED_DIR}"
  sudo mkdir -p "${UNMINED_DIR}"

  TMPDIR=$(mktemp -d)
  wget -q -O "${TMPDIR}/unmined-cli.tgz" "https://unmined.net/download/unmined-cli-linux-x64-${UNMINED_VERSION}/"
  echo "bb77dfb38578dfbfd1289df66026051454f3f8be5cab930740905cc5594e3646  ${TMPDIR}/unmined-cli.tgz" | sha256sum -c -
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
sudo chmod -R 775 "${MAP_DIR}"

sudo tee /etc/caddy/Caddyfile >/dev/null <<EOF
:80 {
    root * ${MAP_DIR}
    file_server
}
EOF

sudo systemctl enable caddy
sudo systemctl restart caddy

