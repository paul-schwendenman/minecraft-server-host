#!/usr/bin/env bash
# build-map-manifests.sh [<world-path>|glob]
# Builds map previews, per-map and per-world manifests, and global index page

set -euo pipefail

WORLD_PATH="${1:-/srv/minecraft-server/world*}"
MAP_ROOT="/srv/minecraft-server/maps"
SERVER_ROOT="/srv/minecraft-server"
UNMINED="/opt/unmined/unmined-cli"
NBT="$HOME/.local/bin/nbt"

# --- Handle wildcard ---
if [[ "$WORLD_PATH" == *"*"* ]]; then
  for w in $WORLD_PATH; do
    [ -d "$w" ] && "$0" "$w"
  done
  exit 0
fi

WORLD_DIR="$WORLD_PATH"
WORLD_NAME=$(basename "$WORLD_DIR")
CONFIG_PATH="$WORLD_DIR/map-config.yml"
MAPS_DIR="$MAP_ROOT/$WORLD_NAME"
mkdir -p "$MAPS_DIR"

if [[ ! -f "$CONFIG_PATH" ]]; then
  echo "⚠️  No config for $WORLD_NAME, skipping"
  exit 0
fi

echo "=== Building manifests for $WORLD_NAME ==="

MAP_COUNT=$(yq e '.maps | length' "$CONFIG_PATH" 2>/dev/null || echo 0)
DIM_JSON="[]"

for (( i=0; i<$MAP_COUNT; i++ )); do
  MAP_NAME=$(yq -r ".maps[$i].name" "$CONFIG_PATH")
  DIMENSION=$(yq -r ".maps[$i].dimension" "$CONFIG_PATH")
  OUT_SUB=$(yq -r ".maps[$i].output_subdir // \"$MAP_NAME\"" "$CONFIG_PATH")
  MAP_OUTPUT="$MAPS_DIR/$OUT_SUB"
  mkdir -p "$MAP_OUTPUT"

  echo "→ Generating preview + manifest for $MAP_NAME ($DIMENSION)"

  LEVEL_DAT="$WORLD_DIR/level.dat"
  SPAWN_X=$($NBT -r --path='Data.SpawnX' "$LEVEL_DAT" 2>/dev/null || echo 0)
  SPAWN_Z=$($NBT -r --path='Data.SpawnZ' "$LEVEL_DAT" 2>/dev/null || echo 0)
  RANGE=64
  AREA="b((${SPAWN_X - RANGE},${SPAWN_Z - RANGE}),(${SPAWN_X + RANGE},${SPAWN_Z + RANGE}))"
  PREVIEW_PATH="$MAP_OUTPUT/preview.png"

  echo "   Rendering preview around spawn (${SPAWN_X}, ${SPAWN_Z})"
  if ! "$UNMINED" image render \
      --world="$WORLD_DIR" \
      --dimension="$DIMENSION" \
      --area="$AREA" \
      --zoom=2 \
      --shadows=3d \
      --trim \
      --output="$PREVIEW_PATH"; then
    echo "⚠️  Failed to render preview for $MAP_NAME"
  fi

  cat >"$MAP_OUTPUT/manifest.json" <<EOF
{
  "world": "$WORLD_NAME",
  "map": "$MAP_NAME",
  "dimension": "$DIMENSION",
  "path": "$WORLD_NAME/$OUT_SUB",
  "last_rendered": "$(date -Iseconds)"
}
EOF

  DIM_JSON=$(echo "$DIM_JSON" | jq --arg name "$MAP_NAME" --arg dim "$DIMENSION" '. + [{"name":$name,"dimension":$dim}]')
done

# --- World-level manifest ---
LEVEL_DAT="$WORLD_DIR/level.dat"
VERSION=$($NBT -r --path='Data.Version.Name' "$LEVEL_DAT" 2>/dev/null || echo "unknown")
LASTPLAYED=$($NBT -r --path='Data.LastPlayed' "$LEVEL_DAT" 2>/dev/null || echo "")
DIFFICULTY_RAW=$($NBT -r --path='Data.Difficulty' "$LEVEL_DAT" 2>/dev/null || echo "")
[[ "$DIFFICULTY_RAW" =~ ^[0-9]+$ ]] || DIFFICULTY_RAW="null"

case "$DIFFICULTY_RAW" in
  0) DIFF_NAME="Peaceful";;
  1) DIFF_NAME="Easy";;
  2) DIFF_NAME="Normal";;
  3) DIFF_NAME="Hard";;
  *) DIFF_NAME="Unknown";;
esac

if [[ "$LASTPLAYED" =~ ^[0-9]+$ ]]; then
  LASTPLAYED_ISO=$(date -u -d "@$((LASTPLAYED/1000))" +"%Y-%m-%dT%H:%M:%SZ")
else
  LASTPLAYED_ISO=""
fi

jq -n \
  --arg world "$WORLD_NAME" \
  --arg version "$VERSION" \
  --argjson maps "$DIM_JSON" \
  --arg rendered "$(date -Iseconds)" \
  --arg last_played "$LASTPLAYED_ISO" \
  --argjson difficulty "$DIFFICULTY_RAW" \
  --arg difficulty_name "$DIFF_NAME" \
  '{
    world: $world,
    version: $version,
    difficulty: $difficulty,
    difficulty_name: $difficulty_name,
    last_played: ($last_played // ""),
    maps: $maps,
    last_rendered: $rendered
  }' > "${MAPS_DIR}/manifest.json"

echo "✅ Created manifest for $WORLD_NAME → ${MAPS_DIR}/manifest.json"

# --- Aggregate + index ---
AGG_FILE="${MAP_ROOT}/world_manifest.json"
INDEX_FILE="${MAP_ROOT}/index.html"

jq -s '.' "${MAP_ROOT}"/*/manifest.json > "$AGG_FILE"
{
  echo "<!DOCTYPE html><html><head><title>Minecraft Maps</title>"
  echo "<style>body{font-family:sans-serif;margin:2rem;}h2{margin-top:1rem}</style>"
  echo "</head><body><h1>Minecraft Worlds</h1>"
  for world_dir in "${MAP_ROOT}"/*/; do
    [[ -d "$world_dir" ]] || continue
    world_name=$(basename "$world_dir")
    echo "<h2>${world_name}</h2><ul>"
    for map_dir in "${world_dir}"*/; do
      [[ -d "$map_dir" ]] || continue
      map_name=$(basename "$map_dir")
      echo "<li><a href='./${world_name}/${map_name}/'>${map_name}</a></li>"
    done
    echo "</ul>"
  done
  echo "</body></html>"
} > "$INDEX_FILE"

echo "🌐 Updated index + aggregate → $INDEX_FILE"
