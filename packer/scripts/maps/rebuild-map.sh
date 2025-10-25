#!/usr/bin/env bash
# rebuild-map.sh <world-path|glob> [--map <name>] [--force] [--non-blocking]
# Renders static maps for Minecraft worlds using uNmINeD
# Respects per-world map-config.yml and skips up-to-date maps

set -euo pipefail

WORLD_PATH="${1:-}"
MAP_FILTER=""
FORCE=false
NONBLOCK=false
MAPS_ROOT="/srv/minecraft-server/maps"
UNMINED="/opt/unmined/unmined-cli"
LOCK_FILE="/tmp/minecraft-map-build.lock"

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

# --- Acquire lock ---
exec 200>"$LOCK_FILE"
if $NONBLOCK; then
  # Try immediately; skip if another rebuild is running
  if ! flock -n 200; then
    echo "⚠️  Another map rebuild is already running — skipping (non-blocking mode)." >&2
    exit 0
  fi
else
  # Wait up to 10 minutes for an existing rebuild to finish
  if ! flock -w 600 200; then
    echo "⚠️  Timeout waiting for map rebuild lock — skipping." >&2
    exit 0
  fi
fi
trap 'rm -f "$LOCK_FILE"' EXIT

# --- Handle wildcard globs ---
if [[ "$WORLD_PATH" == *"*"* ]]; then
  for w in $WORLD_PATH; do
    [ -d "$w" ] && "$0" "$w" ${MAP_FILTER:+--map "$MAP_FILTER"} $($FORCE && echo "--force")
  done
  exit 0
fi

if [[ -z "$WORLD_PATH" || ! -d "$WORLD_PATH" ]]; then
  echo "❌ World path not found: $WORLD_PATH" >&2
  exit 1
fi

CONFIG_PATH="$WORLD_PATH/map-config.yml"
WORLD_NAME=$(basename "$WORLD_PATH")
MAPS_DIR="$MAPS_ROOT/$WORLD_NAME"
mkdir -p "$MAPS_DIR"

if [[ ! -f "$CONFIG_PATH" ]]; then
  echo "⚠️  No config found for $WORLD_NAME at $CONFIG_PATH, skipping"
  exit 0
fi

echo "=== Rebuilding maps for $WORLD_NAME ==="

DEFAULT_ZOUT=$(yq -r '.defaults.zoomout // 2' "$CONFIG_PATH")
DEFAULT_ZIN=$(yq -r '.defaults.zoomin // 1' "$CONFIG_PATH")
DEFAULT_FMT=$(yq -r '.defaults.imageformat // "jpeg"' "$CONFIG_PATH")
DEFAULT_CPUS=$(yq -r '.defaults.chunkprocessors // 2' "$CONFIG_PATH")

MAP_COUNT=$(yq e '.maps | length' "$CONFIG_PATH" 2>/dev/null || echo 0)
if [[ "$MAP_COUNT" -eq 0 ]]; then
  echo "⚠️  No maps defined for $WORLD_NAME"
  exit 0
fi

WORLD_LEVEL="$WORLD_PATH/level.dat"
world_mtime=$(stat -c %Y "$WORLD_LEVEL" 2>/dev/null || echo 0)

for (( i=0; i<$MAP_COUNT; i++ )); do
  MAP_NAME=$(yq -r ".maps[$i].name" "$CONFIG_PATH")
  [[ -n "$MAP_FILTER" && "$MAP_FILTER" != "$MAP_NAME" ]] && continue

  DIM=$(yq -r ".maps[$i].dimension" "$CONFIG_PATH")
  ZOUT=$(yq -r ".maps[$i].zoomout // $DEFAULT_ZOUT" "$CONFIG_PATH")
  ZIN=$(yq -r ".maps[$i].zoomin // $DEFAULT_ZIN" "$CONFIG_PATH")
  OUT_SUB=$(yq -r ".maps[$i].output_subdir // \"$MAP_NAME\"" "$CONFIG_PATH")
  MAP_OUTPUT="$MAPS_DIR/$OUT_SUB"
  mkdir -p "$MAP_OUTPUT"

  MANIFEST_FILE="$MAP_OUTPUT/manifest.json"
  render_needed=true

  if [[ -f "$MANIFEST_FILE" && $FORCE == false ]]; then
    manifest_epoch=$(jq -r '.last_rendered_epoch // 0' "$MANIFEST_FILE" 2>/dev/null || echo 0)
    if (( manifest_epoch >= world_mtime )); then
      echo "🟡 Skipping up-to-date map: $MAP_NAME"
      render_needed=false
    fi
  fi

  if ! $render_needed; then
    continue
  fi

  echo "🔁 Rendering map: $MAP_NAME (dim=$DIM zoom=$ZOUT→$ZIN)"
  BASE_CMD=( "$UNMINED" web render
    --world "$WORLD_PATH"
    --dimension "$DIM"
    --output "$MAP_OUTPUT"
    --zoomout "$ZOUT"
    --zoomin "$ZIN"
    --imageformat "$DEFAULT_FMT"
    --chunkprocessors "$DEFAULT_CPUS"
    --log-level information
  )

  for key in topY bottomY gndxray night shadows; do
    val=$(yq -r ".maps[$i].options.$key // empty" "$CONFIG_PATH")
    [[ -n "$val" && "$val" != "null" ]] && BASE_CMD+=(--$key "$val")
  done

  echo "  • Base: ${BASE_CMD[*]}"
  "${BASE_CMD[@]}"

  RANGE_COUNT=$(yq e ".maps[$i].ranges | length" "$CONFIG_PATH" 2>/dev/null || echo 0)
  if [[ $RANGE_COUNT -gt 0 ]]; then
    for (( j=0; j<$RANGE_COUNT; j++ )); do
      NAME=$(yq -r ".maps[$i].ranges[$j].name" "$CONFIG_PATH")
      CX=$(yq -r ".maps[$i].ranges[$j].center[0]" "$CONFIG_PATH")
      CZ=$(yq -r ".maps[$i].ranges[$j].center[1]" "$CONFIG_PATH")
      RADIUS=$(yq -r ".maps[$i].ranges[$j].radius" "$CONFIG_PATH")
      ZOUT_R=$(yq -r ".maps[$i].ranges[$j].zoomout // $ZOUT" "$CONFIG_PATH")
      ZIN_R=$(yq -r ".maps[$i].ranges[$j].zoomin // $ZIN" "$CONFIG_PATH")

      echo "    → Range: $NAME ($CX,$CZ r=$RADIUS zoom=$ZOUT_R→$ZIN_R)"
      RANGE_CMD=("${BASE_CMD[@]}")
      RANGE_CMD+=(--zoomout "$ZOUT_R" --zoomin "$ZIN_R" \
                  --area "$((CX - RADIUS)),$((CZ - RADIUS)),$((CX + RADIUS)),$((CZ + RADIUS))")
      "${RANGE_CMD[@]}"
    done
  fi

  # --- Update manifest with render time ---
  jq -n \
    --arg world "$WORLD_NAME" \
    --arg map "$MAP_NAME" \
    --arg dim "$DIM" \
    --arg time "$(date -Iseconds)" \
    --argjson epoch "$(date +%s)" \
    '{
      world: $world,
      map: $map,
      dimension: $dim,
      last_rendered: $time,
      last_rendered_epoch: $epoch
    }' > "$MANIFEST_FILE"

done

echo "✅ Completed rebuild for $WORLD_NAME → $MAPS_DIR"
