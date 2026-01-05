#!/usr/bin/env bash
set -euo pipefail

MINECRAFT_HOME="/srv/minecraft-server"
TOUCH_FILE="${MINECRAFT_HOME}/no_one_playing"

# Skip shutdown if SSH sessions exist
if who | grep 'pts/' >/dev/null 2>&1; then
  [ -f "${TOUCH_FILE}" ] && rm -f "${TOUCH_FILE}"
  logger -t autoshutdown "Skipping shutdown: active SSH session(s) detected"
  exit 0
fi

# Check if any minecraft service is running
RUNNING_SERVICES=$(systemctl list-units --type=service --state=running 'minecraft@*.service' --no-legend 2>/dev/null | wc -l)

if [[ "${RUNNING_SERVICES}" -eq 0 ]]; then
  logger -t autoshutdown "No minecraft services running, shutting down now"
  rm -f "${TOUCH_FILE}"
  sudo /sbin/poweroff
  exit 0
fi

# Load shared RCON settings for minecraftctl
# minecraftctl reads RCON_PASSWORD, RCON_HOST, and RCON_PORT from environment variables
if [[ -r /etc/minecraft.env ]]; then
  source /etc/minecraft.env
else
  echo "Error: /etc/minecraft.env not readable" >&2
  exit 1
fi

command -v minecraftctl >/dev/null 2>&1 || {
  echo "Error: minecraftctl not found" >&2
  exit 1
}

OUTPUT=$(minecraftctl rcon send "list" || true)
COUNT=$(echo "$OUTPUT" | sed -n 's/.*There are \([0-9]*\).*/\1/p' | head -1)
COUNT=${COUNT:-0}
logger -t autoshutdown "RCON reports $COUNT players online"

if [[ "${COUNT}" -eq 0 ]]; then
  if [ -f "${TOUCH_FILE}" ]; then
    logger -t autoshutdown "No players - second check, shutting down now"
    rm -f "${TOUCH_FILE}"
    sudo /sbin/poweroff
    exit 0
  else
    logger -t autoshutdown "No players — first check, will shut down next time"
    touch "${TOUCH_FILE}"
  fi
else
  [ -f "${TOUCH_FILE}" ] && rm -f "${TOUCH_FILE}"
  logger -t autoshutdown "Players online — skipping shutdown"
fi
