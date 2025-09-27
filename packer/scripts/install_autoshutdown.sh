#!/usr/bin/env bash
set -euxo pipefail

MINECRAFT_HOME="/srv/minecraft-server"
MINECRAFT_USER="minecraft"

sudo tee /usr/local/bin/autoshutdown.sh >/dev/null <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

MINECRAFT_HOME="/srv/minecraft-server"
TOUCH_FILE="${MINECRAFT_HOME}/no_one_playing"

# If anyone is SSH'd in, don't shutdown
if who | grep 'pts/' >/dev/null 2>&1; then
  [ -f "${TOUCH_FILE}" ] && rm -f "${TOUCH_FILE}"
  logger -t autoshutdown "Skipping shutdown: active SSH session(s) detected"
  exit 0
fi

COUNT=0

# Prefer RCON if configured (requires server.properties with rcon.port & rcon.password)
if command -v mcrcon >/dev/null 2>&1 && [ -n "${RCON_PASSWORD:-}" ]; then
  # Expects RCON_PASSWORD env (set via systemd Environment= or /etc/environment)
  OUTPUT=$(mcrcon -H 127.0.0.1 -p "$RCON_PASSWORD" list || true)
  # format: "There are 0 of a max 20 players online: "
  COUNT=$(echo "$OUTPUT" | awk -F' ' '/There are/ {print $3}' || echo "0")
  logger -t autoshutdown "Queried via RCON: $COUNT players online"
else
  # Fallback: poke 'list' into screen, then parse latest.log
  # (works for the single-world unit; @-template would need per-world dir)
  if screen -list | grep -q minecraft; then
    screen -S minecraft -p 0 -X stuff "list^M"
    sleep 5
  fi
  LAST_LINE="$(tail -n 1 ${MINECRAFT_HOME}/logs/latest.log || true)"
  # crude parse
  COUNT=$(echo "$LAST_LINE" | sed -n 's/.*There are \([0-9]\+\) of a max.*/\1/p')
  COUNT=${COUNT:-0}
  logger -t autoshutdown "Parsed logs: $COUNT players online"
fi

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
EnvironmentFile=-/etc/default/minecraft
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
