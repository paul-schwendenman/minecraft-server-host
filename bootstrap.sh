#!/usr/bin/env bash

set -o nounset
set -o errexit
set -o pipefail

MINECRAFT_HOME="/srv/minecraft-server"
MINECRAFT_USER="minecraft"
MINECRAFT_GROUP="minecraft"

# Setup minecraft user
sudo adduser --system --home "${MINECRAFT_HOME}" "${MINECRAFT_USER}"
sudo addgroup --system "${MINECRAFT_GROUP}"
sudo adduser "${MINECRAFT_USER}" "${MINECRAFT_GROUP}"
sudo chown -R "${MINECRAFT_USER}.${MINECRAFT_GROUP}" "${MINECRAFT_HOME}"

# Install java
sudo apt update
sudo apt install -y openjdk-11-jdk-headless

# Download server
wget https://launcher.mojang.com/v1/objects/d0d0fe2b1dc6ab4c65554cb734270872b72dadd6/server.jar
echo "942256f0bfec40f2331b1b0c55d7a683b86ee40e51fa500a2aa76cf1f1041b38  server.jar" | shasum -a256 -c -
sudo mv server.jar "${MINECRAFT_HOME}/minecraft_server.jar"

# Install EULA
sudo mv eula.txt "${MINECRAFT_HOME}"

# Install systemd service
sudo cp minecraft.service /etc/systemd/system/minecraft.service

# Start minecraft
sudo service minecraft start
