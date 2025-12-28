#!/usr/bin/env bash
set -euxo pipefail

REPO="paul-schwendenman/minecraft-server-host"

# --- minecraftctl ---
if ! command -v minecraftctl >/dev/null 2>&1; then
  echo "[*] Installing minecraftctl"
  MINECRAFTCTL_VERSION="${MINECRAFTCTL_VERSION:-latest}"

  if [ "$MINECRAFTCTL_VERSION" = "latest" ]; then
    # Get latest release tag
    TAG=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
  else
    TAG="minecraftctl-${MINECRAFTCTL_VERSION}"
  fi

  cd /tmp
  wget -q "https://github.com/${REPO}/releases/download/${TAG}/minecraftctl-linux-amd64" -O minecraftctl-linux-amd64

  # Download and verify checksum
  wget -q "https://github.com/${REPO}/releases/download/${TAG}/checksums.txt" -O checksums.txt
  sha256sum -c --ignore-missing checksums.txt

  sudo install -m 755 minecraftctl-linux-amd64 /usr/local/bin/minecraftctl

  rm -f minecraftctl-linux-amd64 checksums.txt
fi

# --- man pages ---
if [ ! -f /usr/local/share/man/man1/minecraftctl.1 ]; then
  echo "[*] Installing minecraftctl man pages"
  MINECRAFTCTL_VERSION="${MINECRAFTCTL_VERSION:-latest}"

  if [ "$MINECRAFTCTL_VERSION" = "latest" ]; then
    TAG=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
  else
    TAG="minecraftctl-${MINECRAFTCTL_VERSION}"
  fi

  cd /tmp
  if wget -q "https://github.com/${REPO}/releases/download/${TAG}/minecraftctl-man.tar.gz" -O minecraftctl-man.tar.gz 2>/dev/null; then
    tar -xzf minecraftctl-man.tar.gz
    sudo mkdir -p /usr/local/share/man/man1
    sudo install -m 644 man/man1/*.1 /usr/local/share/man/man1/
    rm -rf minecraftctl-man.tar.gz man/
    echo "[*] Man pages installed"
  else
    echo "[!] Man pages not available for this release (skipping)"
  fi
fi

# --- bash completion ---
if [ ! -f /etc/bash_completion.d/minecraftctl ]; then
  echo "[*] Installing minecraftctl bash completion"
  MINECRAFTCTL_VERSION="${MINECRAFTCTL_VERSION:-latest}"

  if [ "$MINECRAFTCTL_VERSION" = "latest" ]; then
    TAG=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
  else
    TAG="minecraftctl-${MINECRAFTCTL_VERSION}"
  fi

  cd /tmp
  if wget -q "https://github.com/${REPO}/releases/download/${TAG}/minecraftctl.bash" -O minecraftctl.bash 2>/dev/null; then
    sudo mkdir -p /etc/bash_completion.d
    sudo install -m 644 minecraftctl.bash /etc/bash_completion.d/minecraftctl
    rm -f minecraftctl.bash
    echo "[*] Bash completion installed"
  else
    echo "[!] Bash completion not available for this release (skipping)"
  fi
fi

