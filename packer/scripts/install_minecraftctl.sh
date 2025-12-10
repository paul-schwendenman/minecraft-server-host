#!/usr/bin/env bash
set -euxo pipefail

# --- minecraftctl ---
if ! command -v minecraftctl >/dev/null 2>&1; then
  echo "[*] Installing minecraftctl"
  MINECRAFTCTL_VERSION="${MINECRAFTCTL_VERSION:-latest}"
  
  if [ "$MINECRAFTCTL_VERSION" = "latest" ]; then
    # Get latest release tag
    TAG=$(curl -s https://api.github.com/repos/paul-schwendenman/minecraft-server-host/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
  else
    TAG="minecraftctl-${MINECRAFTCTL_VERSION}"
  fi
  
  cd /tmp
  wget -q "https://github.com/paul-schwendenman/minecraft-server-host/releases/download/${TAG}/minecraftctl-linux-amd64" -O minecraftctl-linux-amd64
  
  # Download and verify checksum
  wget -q "https://github.com/paul-schwendenman/minecraft-server-host/releases/download/${TAG}/checksums.txt" -O checksums.txt
  sha256sum -c --ignore-missing checksums.txt
  
  sudo mv minecraftctl-linux-amd64 /usr/local/bin/minecraftctl
  sudo chmod 0755 /usr/local/bin/minecraftctl
  sudo chown root:root /usr/local/bin/minecraftctl
  
  rm -f checksums.txt
fi

