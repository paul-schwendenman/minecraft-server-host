#!/usr/bin/env bash
set -euxo pipefail

MINECRAFT_HOME="/srv/minecraft-server"
MINECRAFT_USER="minecraft"
MINECRAFT_GROUP="minecraft"

# System user + dirs
id -u "${MINECRAFT_USER}" >/dev/null 2>&1 || sudo useradd -m -r -s /bin/bash "${MINECRAFT_USER}"
getent group "${MINECRAFT_GROUP}" >/dev/null 2>&1 || sudo groupadd -r "${MINECRAFT_GROUP}"
sudo usermod -a -G "${MINECRAFT_GROUP}" "${MINECRAFT_USER}"

sudo mkdir -p "${MINECRAFT_HOME}"
sudo chown -R "${MINECRAFT_USER}:${MINECRAFT_GROUP}" "${MINECRAFT_HOME}"

# Service template (per-world)
sudo tee /etc/systemd/system/minecraft@.service >/dev/null <<'EOF'
[Unit]
Description=Minecraft Server %i
After=network-online.target
Wants=network-online.target

[Service]
WorkingDirectory=/srv/minecraft-server/%i
User=minecraft
Group=minecraft
UMask=002

# Harden a bit
PrivateTmp=true
ProtectSystem=full
ProtectHome=true
NoNewPrivileges=true

# Start
Type=forking
ExecStart=/usr/bin/screen -h 2048 -dmS mc-%i /usr/bin/java -Xms1536M -Xmx1536M -jar server.jar nogui
ExecReload=/usr/bin/screen -S mc-%i -p 0 -X stuff "reload^M"

# Graceful shutdown; map rebuild hook added via drop-in
ExecStop=/usr/bin/screen -S mc-%i -p 0 -X stuff "say SERVER SHUTTING DOWN. Saving map...^M"
ExecStop=/usr/bin/screen -S mc-%i -p 0 -X stuff "save-all^M"
ExecStop=/usr/bin/screen -S mc-%i -p 0 -X stuff "stop^M"
TimeoutStopSec=30

Restart=on-failure
RestartSec=20

[Install]
WantedBy=multi-user.target
EOF

# Compatibility single-world unit
sudo tee /etc/systemd/system/minecraft.service >/dev/null <<'EOF'
[Unit]
Description=minecraft-server
After=network-online.target
Wants=network-online.target

[Service]
WorkingDirectory=/srv/minecraft-server
User=minecraft
Group=minecraft
UMask=002

PrivateTmp=true
ProtectSystem=full
ProtectHome=true
NoNewPrivileges=true

Type=forking
ExecStart=/usr/bin/screen -h 2048 -dmS minecraft /usr/bin/java -Xms1536M -Xmx1536M -jar server.jar nogui
ExecReload=/usr/bin/screen -p 0 -S minecraft -X eval 'stuff "reload"\\015'
ExecStop=/usr/bin/screen -p 0 -S minecraft -X eval 'stuff "say SERVER SHUTTING DOWN. Saving map..."\\015'
ExecStop=/usr/bin/screen -p 0 -S minecraft -X eval 'stuff "save-all"\\015'
ExecStop=/usr/bin/screen -p 0 -S minecraft -X eval 'stuff "stop"\\015'
TimeoutStopSec=30

Restart=on-failure
RestartSec=20

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
