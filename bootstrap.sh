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
wget https://launcher.mojang.com/v1/objects/d0d0fe2b1dc6ab4c65554cb734270872b72dadd6/server.jar minecraft_server.1.14.3.jar
echo "942256f0bfec40f2331b1b0c55d7a683b86ee40e51fa500a2aa76cf1f1041b38 minecraft_server.1.14.3.jar" | shasum -a256 -c -
sudo cp minecraft_server.1.14.3.jar "${MINECRAFT_HOME}/minecraft_server.jar"

# Accept EULA
sudo -u "${MINECRAFT_USER}" tee "${MINECRAFT_HOME}/eula.txt" > /dev/null << EOF
eula=true
EOF

# Install systemd service
sudo tee /etc/systemd/system/minecraft.service > /dev/null << EOF
[Unit]
Description=minecraft-server
After=network.target

[Service]
WorkingDirectory=/srv/minecraft-server
User=minecraft
Group=minecraft

PrivateUsers=true
# Users Database is not available for within the unit, only root and minecraft is available, everybody else is nobody
ProtectSystem=full
# Read only mapping of /usr /boot and /etc
ProtectHome=true
# /home, /root and /run/user seem to be empty from within the unit. It is recommended to enable this setting for all long-running services (in particular network-facing ones).
ProtectKernelTunables=true
# /proc/sys, /sys, /proc/sysrq-trigger, /proc/latency_stats, /proc/acpi, /proc/timer_stats, /proc/fs and /proc/irq will be read-only within the unit. It is recommended to turn this on for most services.
# Implies MountFlags=slave
ProtectKernelModules=true
# Block module system calls, also /usr/lib/modules. It is recommended to turn this on for most services that do not need special file systems or extra kernel modules to work
# Implies NoNewPrivileges=yes
ProtectControlGroups=true
# It is hence recommended to turn this on for most services.
# Implies MountAPIVFS=yes

Type=forking
Restart=on-failure
RestartSec=20 5
ExecStart=/usr/bin/screen -h 2048 -dmS minecraft /usr/bin/java -Xms1536M -Xmx1536M -jar minecraft_server.jar nogui

[Install]
WantedBy=multi-user.target
EOF

# Start minecraft
sudo service minecraft start
