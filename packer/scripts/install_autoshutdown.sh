#!/usr/bin/env bash
set -euxo pipefail

MINECRAFT_HOME="/srv/minecraft-server"
MINECRAFT_USER="minecraft"

sudo tee /usr/local/bin/autoshutdown.sh >/dev/null <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

MINECRAFT_HOME="/srv/minecraft-server"
TOUCH_FILE="${MINECRAFT_HOME}/no_one_playing"

# Load shared RCON settings
source /etc/minecraft.env

# If anyone is SSH’d in, skip shutdown
if who | grep 'pts/' >/dev/null 2>&1; then
  [ -f "${TOUCH_FILE}" ] && rm -f "${TOUCH_FILE}"
  logger -t autoshutdown "Skipping shutdown: active SSH session(s) detected"
  exit 0
fi

# Query via RCON
OUTPUT=$(mcrcon -H 127.0.0.1 -P "$RCON_PORT" -p "$RCON_PASSWORD" list || true)
COUNT=$(echo "$OUTPUT" | awk -F' ' '/There are/ {print $3}' || echo "0")
logger -t autoshutdown "RCON reports $COUNT players online"

if [[ "${COUNT}" -eq 0 ]]; then
  if [ -f "${TOUCH_FILE}" ]; then
    logger -t autoshutdown "No players twice in a row — shutting down now"
    rm -f "${TOUCH_FILE}"
    sudo /sbin/poweroff
  else
    logger -t autoshutdown "No players — first check, will shut down if still empty next time"
    touch "${TOUCH_FILE}"
  fi
else
  [ -f "${TOUCH_FILE}" ] && rm -f "${TOUCH_FILE}"
  logger -t autoshutdown "Players online — skipping shutdown"
fi
EOF
sudo chmod 0755 /usr/local/bin/autoshutdown.sh
sudo chown root:root /usr/local/bin/autoshutdown.sh

sudo tee /etc/sudoers.d/minecraft-shutdown >/dev/null <<EOF
minecraft ALL=(root) NOPASSWD: /sbin/poweroff
EOF
sudo chmod 0440 /etc/sudoers.d/minecraft-shutdown

sudo tee /etc/systemd/system/autoshutdown.service >/dev/null <<'EOF'
[Unit]
Description=Check for idle Minecraft server

[Service]
Type=oneshot
User=minecraft
EnvironmentFile=-/etc/minecraft.env
ExecStart=/usr/local/bin/autoshutdown.sh
EOF

sudo tee /etc/systemd/system/autoshutdown.timer >/dev/null <<'EOF'
[Unit]
Description=Run Minecraft idle check every 5 minutes

[Timer]
OnBootSec=5min
OnUnitActiveSec=5min
Unit=autoshutdown.service

[Install]
WantedBy=timers.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable autoshutdown.timer
