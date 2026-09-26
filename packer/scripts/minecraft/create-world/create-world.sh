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

WORLDS_DIR="/srv/minecraft-server"

# Enable the timers of every other world on the volume, so a new instance keeps
# backing up all worlds, not just the one Terraform names. A world that can't be
# registered (e.g. never started, so no level.dat) is skipped with a warning.
register_other_worlds() {
  local props other
  for props in "${WORLDS_DIR}"/*/server.properties; do
    [[ -e "$props" ]] || continue
    other=$(basename "$(dirname "$props")")
    [[ "$other" == "$WORLD" ]] && continue
    echo "[userdata] Registering world '$other'."
    minecraftctl world register "$other" || echo "[userdata] Could not register '$other', skipping." >&2
  done
}

# Check if world already exists on EBS volume
if [ -d "${WORLDS_DIR}/$WORLD" ]; then
  echo "[userdata] World '$WORLD' already exists on EBS. Registering it."
  minecraftctl world register "$WORLD"
  sudo chown -R minecraft:minecraft "${WORLDS_DIR}/$WORLD"
  register_other_worlds
  exit 0
fi

JAR_PATH="/opt/minecraft/jars/minecraft_server_${VERSION}.jar"

if [[ ! -f "$JAR_PATH" ]]; then
  echo "Error: Minecraft server jar for version $VERSION not found at $JAR_PATH" >&2
  exit 1
fi

# Build minecraftctl command
# Note: this doesn't start the world. minecraft-active.service starts whichever
# world the ActiveWorld instance tag names once cloud-init has finished.
CMD=("minecraftctl" "world" "create" "$WORLD" "--version" "$VERSION")

if [[ -n "$SEED" ]]; then
  CMD+=("--seed" "$SEED")
fi

# Execute minecraftctl
"${CMD[@]}"

sudo chown -R minecraft:minecraft "${WORLDS_DIR}/$WORLD"
register_other_worlds
