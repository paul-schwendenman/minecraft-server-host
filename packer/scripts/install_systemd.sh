#!/usr/bin/env bash
set -euxo pipefail

# Create minecraft system user
sudo useradd -m -r -s /bin/bash minecraft || true

# Create base dir
sudo mkdir -p /srv/minecraft
sudo chown minecraft:minecraft /srv/minecraft

# Systemd template unit
sudo tee /etc/systemd/system/minecraft@.service > /dev/null <<'EOF'
[Unit]
Description=Minecraft Server %i
After=network.target

[Service]
WorkingDirectory=/srv/minecraft/%i
User=minecraft
ExecStart=/usr/bin/screen -DmS mc-%i /usr/bin/java -Xmx4G -Xms1G -jar server.jar nogui
ExecStop=/usr/bin/screen -S mc-%i -p 0 -X stuff "say Server shutting down^Mstop^M"
Restart=on-failure

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
