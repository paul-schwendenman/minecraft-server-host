#!/usr/bin/env bash
# rebuild-map.sh <world-path|glob> [--map <name>] [--force] [--non-blocking]
# Wrapper around minecraftctl map build

set -euo pipefail

WORLD_PATH="${1:-}"
MAP_FILTER=""
FORCE=false
NONBLOCK=false

# --- Parse optional args ---
shift || true
while [[ $# -gt 0 ]]; do
  case "$1" in
    --map) MAP_FILTER="$2"; shift 2 ;;
    --force) FORCE=true; shift ;;
    --non-blocking) NONBLOCK=true; shift ;;
    *) echo "Unknown argument: $1" >&2; exit 1 ;;
  esac
done

# --- Detect glob mode early ---
IS_GLOB=false
if [[ "$WORLD_PATH" == *"*"* ]]; then
  IS_GLOB=true
fi

# --- Handle wildcard globs ---
if $IS_GLOB; then
  for w in $WORLD_PATH; do
    if [[ -d "$w" ]]; then
      WORLD_NAME=$(basename "$w")
      CMD=("minecraftctl" "map" "build" "$WORLD_NAME")
      [[ -n "$MAP_FILTER" ]] && CMD+=("--map" "$MAP_FILTER")
      $FORCE && CMD+=("--force")
      $NONBLOCK && CMD+=("--non-blocking")
      "${CMD[@]}" || true  # Continue on error for glob processing
    fi
  done
  exit 0
fi

if [[ -z "$WORLD_PATH" || ! -d "$WORLD_PATH" ]]; then
  echo "❌ World path not found: $WORLD_PATH" >&2
  exit 1
fi

# Extract world name from path
WORLD_NAME=$(basename "$WORLD_PATH")

# Build minecraftctl command
CMD=("minecraftctl" "map" "build" "$WORLD_NAME")

if [[ -n "$MAP_FILTER" ]]; then
  CMD+=("--map" "$MAP_FILTER")
fi

if $FORCE; then
  CMD+=("--force")
fi

if $NONBLOCK; then
  CMD+=("--non-blocking")
fi

# Execute minecraftctl
exec "${CMD[@]}"
