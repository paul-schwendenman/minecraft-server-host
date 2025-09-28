#!/usr/bin/env bash
set -euxo pipefail

sudo tee /etc/systemd/system/minecraft-maprefresh.service > /dev/null <<'EOF'
[Unit]
Description=Save world and rebuild uNmINeD map
After=network.target

[Service]
Type=oneshot
User=minecraft
EnvironmentFile=-/etc/minecraft.env
ExecStartPre=/usr/bin/mcrcon -H 127.0.0.1 -P ${RCON_PORT} -p ${RCON_PASSWORD} "say Saving map..."
ExecStart=/usr/bin/mcrcon -H 127.0.0.1 -P ${RCON_PORT} -p ${RCON_PASSWORD} save-all
ExecStartPost=/usr/local/bin/rebuild-map.sh /srv/minecraft-server/*/world
EOF

sudo tee /etc/systemd/system/minecraft-maprefresh.timer > /dev/null <<'EOF'
[Unit]
Description=Run Minecraft map refresh periodically

[Timer]
OnBootSec=1min
OnUnitActiveSec=15min
Unit=minecraft-maprefresh.service

[Install]
WantedBy=timers.target
EOF

# Enable and start timer
sudo systemctl daemon-reload
sudo systemctl enable --now minecraft-maprefresh.timer
