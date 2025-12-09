#!/usr/bin/env bash
set -euxo pipefail

MOUNT_POINT="${MOUNT_POINT:-/srv/minecraft-server}"
CONFIG_FILE="$MOUNT_POINT/minecraft.env"

if [ -f "$CONFIG_FILE" ]; then
  echo "Reusing existing $CONFIG_FILE"
  rm -f /etc/minecraft.env
  ln -s "$CONFIG_FILE" /etc/minecraft.env
elif [ -f /etc/minecraft.env ]; then
  echo "Migrating /etc/minecraft.env -> $CONFIG_FILE"
  mv /etc/minecraft.env "$CONFIG_FILE"
  ln -s "$CONFIG_FILE" /etc/minecraft.env
else
  echo "Generating new $CONFIG_FILE"
  cat > "$CONFIG_FILE" <<EOF
RCON_PASSWORD=$(openssl rand -hex 16)
RCON_PORT=25575
EOF
  ln -s "$CONFIG_FILE" /etc/minecraft.env
fi
