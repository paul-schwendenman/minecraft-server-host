#!/usr/bin/env bash
# build-map-manifests.sh [<world-path>|glob]
# Wrapper around minecraftctl map manifest and map index

set -euo pipefail

WORLD_PATH="${1:-/srv/minecraft-server/world*}"

# --- Handle wildcard ---
if [[ "$WORLD_PATH" == *"*"* ]]; then
  WORLDS=()
  for w in $WORLD_PATH; do
    if [[ -d "$w" ]]; then
      WORLDS+=("$(basename "$w")")
    fi
  done
  
  if [[ ${#WORLDS[@]} -eq 0 ]]; then
    echo "⚠️  No worlds found matching pattern: $WORLD_PATH"
    exit 0
  fi
  
  # Build manifests for all worlds
  for world_name in "${WORLDS[@]}"; do
    echo "=== Building manifests for $world_name ==="
    minecraftctl map manifest "$world_name" || true  # Continue on error
  done
  
  # Update aggregate index after processing all worlds
  if [[ ${#WORLDS[@]} -gt 0 ]]; then
    echo "🌐 Updating aggregate index..."
    minecraftctl map index || true
  fi
  
  exit 0
fi

# Single world path
if [[ ! -d "$WORLD_PATH" ]]; then
  echo "⚠️  World path not found: $WORLD_PATH"
  exit 0
fi

WORLD_NAME=$(basename "$WORLD_PATH")

# Build manifest for this world and update index
echo "=== Building manifests for $WORLD_NAME ==="
minecraftctl map manifest "$WORLD_NAME" --update-index
