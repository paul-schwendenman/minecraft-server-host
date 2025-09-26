#!/usr/bin/env bash
set -euxo pipefail

sudo apt-get update
sudo apt-get upgrade -y

# Java (Temurin / OpenJDK 21 for MC 1.20+)
sudo apt-get install -y openjdk-21-jre-headless

# Tools
sudo apt-get install -y screen unzip wget curl python3-pip git

# Python mcstatus
pip3 install --user mcstatus

# mcrcon
MCRCON_VERSION=0.8.1
wget -q https://github.com/Tiiffi/mcrcon/archive/refs/tags/v${MCRCON_VERSION}.tar.gz -O /tmp/mcrcon.tar.gz
tar -xzf /tmp/mcrcon.tar.gz -C /tmp
cd /tmp/mcrcon-${MCRCON_VERSION}
gcc mcrcon.c -o mcrcon
sudo mv mcrcon /usr/local/bin/

# unmined-cli
wget -q https://unmined.net/downloads/unmined-linux.zip -O /tmp/unmined.zip
sudo unzip -o /tmp/unmined.zip -d /usr/local/unmined
