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

# --- minecraftctl ---
if ! command -v minecraftctl >/dev/null 2>&1; then
  echo "[*] Installing minecraftctl"
  MINECRAFTCTL_VERSION="${MINECRAFTCTL_VERSION:-latest}"
  
  if [ "$MINECRAFTCTL_VERSION" = "latest" ]; then
    # Get latest release tag
    TAG=$(curl -s https://api.github.com/repos/paul/minecraft-server-host/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
  else
    TAG="minecraftctl-${MINECRAFTCTL_VERSION}"
  fi
  
  cd /tmp
  wget -q "https://github.com/paul/minecraft-server-host/releases/download/${TAG}/minecraftctl-linux-amd64" -O minecraftctl-linux-amd64
  
  # Download and verify checksum
  wget -q "https://github.com/paul/minecraft-server-host/releases/download/${TAG}/checksums.txt" -O checksums.txt
  sha256sum -c --ignore-missing checksums.txt
  
  sudo mv minecraftctl-linux-amd64 /usr/local/bin/minecraftctl
  sudo chmod 0755 /usr/local/bin/minecraftctl
  sudo chown root:root /usr/local/bin/minecraftctl
  
  rm -f checksums.txt
fi
