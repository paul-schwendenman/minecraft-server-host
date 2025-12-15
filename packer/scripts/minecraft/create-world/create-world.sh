#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 2 ]]; then
  echo "Usage: $0 <world-name> <jar-version> [seed]"
  echo "Example: $0 vanilla-121 1.21.1 8675309"
  exit 1
fi

WORLD="$1"
VERSION="$2"
SEED="${3:-}"

# Check if world already exists on EBS volume
if [ -d "/srv/minecraft-server/$WORLD" ]; then
  echo "[userdata] World '$WORLD' already exists on EBS. Registering it."
  minecraftctl world register "$WORLD"
  sudo chown -R minecraft:minecraft "/srv/minecraft-server/$WORLD"
  exit 0
fi

JAR_PATH="/opt/minecraft/jars/minecraft_server_${VERSION}.jar"

if [[ ! -f "$JAR_PATH" ]]; then
  echo "Error: Minecraft server jar for version $VERSION not found at $JAR_PATH" >&2
  exit 1
fi

# Build minecraftctl command
# Note: minecraftctl world create enables and starts the systemd service by default
# (unless --no-systemd is passed, which we don't do here)
CMD=("minecraftctl" "world" "create" "$WORLD" "--version" "$VERSION")

if [[ -n "$SEED" ]]; then
  CMD+=("--seed" "$SEED")
fi

# Execute minecraftctl
"${CMD[@]}"

sudo chown -R minecraft:minecraft "/srv/minecraft-server/$WORLD"
