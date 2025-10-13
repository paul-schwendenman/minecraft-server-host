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
pip3 install --user mcstatus nbtlib
sudo -u minecraft pip3 install --user nbtlib

# mcrcon (build from source—no external runtime deps)
MCRCON_VERSION=0.7.2
cd /tmp
wget -q "https://github.com/Tiiffi/mcrcon/archive/refs/tags/v${MCRCON_VERSION}.tar.gz" -O mcrcon.tar.gz
echo "1743b25a2d031b774e805f4011cb7d92010cb866e3b892f5dfc5b42080973270  /tmp/mcrcon.tar.gz" | sha256sum -c -
tar -xzf mcrcon.tar.gz
cd "mcrcon-${MCRCON_VERSION}"
make
sudo make install

# Install AWS CLI v2
curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "/tmp/awscliv2.zip"
# echo "bf1150ac34672ebec370f724578c85f6a49f72390670022a06b9cf0f68b6d36e  /tmp/awscliv2.zip" | sha256sum -c -
unzip -q /tmp/awscliv2.zip -d /tmp
sudo /tmp/aws/install
rm -rf /tmp/aws /tmp/awscliv2.zip
