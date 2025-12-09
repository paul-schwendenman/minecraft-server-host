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
exec "${CMD[@]}"
