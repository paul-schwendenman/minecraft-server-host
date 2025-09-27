#!/usr/bin/env bash
set -euxo pipefail

# Base deps
sudo add-apt-repository -y universe
sudo apt-get update -qq
sudo DEBIAN_FRONTEND=noninteractive apt-get upgrade -y -qq
sudo apt-get install -y -qq \
  openjdk-21-jre-headless \
  screen unzip wget curl ca-certificates \
  python3-pip git build-essential jq

# Python tool (optional; nice for local checks)
pip3 install --user mcstatus

# mcrcon (build from source—no external runtime deps)
MCRCON_VERSION=0.7.2
cd /tmp
wget -q "https://github.com/Tiiffi/mcrcon/archive/refs/tags/v${MCRCON_VERSION}.tar.gz" -O mcrcon.tar.gz
tar -xzf mcrcon.tar.gz
cd "mcrcon-${MCRCON_VERSION}"
make
sudo make install
