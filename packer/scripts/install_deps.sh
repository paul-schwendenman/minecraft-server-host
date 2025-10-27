#!/usr/bin/env bash
set -euxo pipefail

# --- Base OS packages ---
sudo add-apt-repository -y universe
sudo apt-get update -qq
sudo DEBIAN_FRONTEND=noninteractive apt-get upgrade -y -qq
sudo apt-get install -y -qq \
  openjdk-21-jre-headless \
  screen unzip wget curl ca-certificates \
  python3-pip git build-essential jq xfsprogs

# --- Python tools (optional) ---
pip3 install --user --upgrade mcstatus nbtlib

# --- mcrcon ---
MCRCON_VERSION="0.7.2"
if ! command -v mcrcon >/dev/null 2>&1; then
  cd /tmp
  wget -q "https://github.com/Tiiffi/mcrcon/archive/refs/tags/v${MCRCON_VERSION}.tar.gz" -O mcrcon.tar.gz
  echo "1743b25a2d031b774e805f4011cb7d92010cb866e3b892f5dfc5b42080973270  mcrcon.tar.gz" | sha256sum -c -
  tar -xzf mcrcon.tar.gz
  cd "mcrcon-${MCRCON_VERSION}"
  make
  sudo make install
fi

# --- AWS CLI v2 ---
if ! command -v aws >/dev/null 2>&1; then
  echo "[*] Installing AWS CLI v2 with GPG verification"

  TMPDIR=$(mktemp -d)
  cd "$TMPDIR"

  # Fetch the CLI package and signature
  curl -s -O https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip
  curl -s -O https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip.sig
  gpg --import /tmp/scripts/deps/awscliv2.pub

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
