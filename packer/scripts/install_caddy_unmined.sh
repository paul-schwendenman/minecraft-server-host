#!/usr/bin/env bash
set -euxo pipefail

MINECRAFT_GROUP="minecraft"
UNMINED_DIR="/opt/unmined"
MAP_DIR="/var/www/map"


# -- Install nbt cli --
sudo -u minecraft pip3 install --user nbtlib

# --- uNmINeD CLI (grab latest linux-x64 dev build) ---
if [[ ! -x "${UNMINED_DIR}/unmined-cli" ]]; then
  sudo rm -rf "${UNMINED_DIR}"
  sudo mkdir -p "${UNMINED_DIR}"

  wget -O /tmp/unmined-cli.tgz "https://unmined.net/download/unmined-cli-linux-x64-dev/"
  echo "9509f4aac9f8efac698931bd71f057bd3dbf8aaa717f6f158dc69ae0be33cfca  /tmp/unmined-cli.tgz" | sha256sum -c -
  tar -xzf /tmp/unmined-cli.tgz -C /tmp

  EXTRACTED_DIR=$(find /tmp -maxdepth 1 -type d -name "unmined-cli_*_linux-x64" | head -n1)
  if [[ -z "$EXTRACTED_DIR" ]]; then
    echo "Could not find extracted unmined directory" >&2
    exit 1
  fi

  sudo mv "$EXTRACTED_DIR"/* "${UNMINED_DIR}/"
  sudo chmod +x "${UNMINED_DIR}/unmined-cli"
fi

# --- Caddy (official repo) ---
if ! command -v caddy >/dev/null 2>&1; then
  sudo apt-get install -y -qq debian-keyring debian-archive-keyring apt-transport-https
  curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' \
    | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
  curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' \
    | sudo tee /etc/apt/sources.list.d/caddy-stable.list
  sudo chmod o+r /usr/share/keyrings/caddy-stable-archive-keyring.gpg
  sudo chmod o+r /etc/apt/sources.list.d/caddy-stable.list
  sudo apt-get update -qq
  sudo apt-get install -y -qq caddy
fi

# Map dir + Caddyfile
sudo mkdir -p "${MAP_DIR}"
sudo chown -R root:${MINECRAFT_GROUP} "${MAP_DIR}"
sudo chmod -R 775 "${MAP_DIR}"

sudo tee /etc/caddy/Caddyfile >/dev/null <<EOF
:80 {
    root * ${MAP_DIR}
    file_server
}
EOF

sudo systemctl enable caddy
sudo systemctl restart caddy

# --- Map rebuild script (callable for any world path) ---
sudo tee /usr/local/bin/rebuild-map.sh >/dev/null <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

WORLD_PATH="${1:-/srv/minecraft-server/world}"
shift || true   # remove world path from args, leave any others

# --- optional args: dimensions, zoom levels ---
declare -a DIM_FILTER=()
ZOOMOUT=6
ZOOMIN=4

while [[ $# -gt 0 ]]; do
  case "$1" in
    overworld|nether|end)
      DIM_FILTER+=("$1")
      ;;
    --zoomout=*)
      ZOOMOUT="${1#*=}"
      ;;
    --zoomin=*)
      ZOOMIN="${1#*=}"
      ;;
    *)
      echo "Unknown argument: $1" >&2
      exit 1
      ;;
  esac
  shift
done

# If WORLD_PATH is a glob, expand it and recurse
if [[ "$WORLD_PATH" == *"*"* ]]; then
  for w in $WORLD_PATH; do
    [ -d "$w" ] && "$0" "$w"
  done
  exit 0
fi

MAP_ROOT="/var/www/map"
UNMINED="/opt/unmined/unmined-cli"

if [[ ! -x "${UNMINED}" ]]; then
  echo "Error: unmined-cli not installed at ${UNMINED}" >&2
  exit 1
fi

# World name is parent directory of 'world', e.g. /srv/minecraft-server/default/world
WORLD_NAME=$(basename "$(dirname "$WORLD_PATH")")
WORLD_BASE="/srv/minecraft-server/${WORLD_NAME}"
MAP_DIR="${MAP_ROOT}/${WORLD_NAME}"

echo "Rebuilding maps for world '${WORLD_NAME}' → ${MAP_DIR}"
mkdir -p "${MAP_DIR}"

# --- Render dimensions ---
# --- Render dimensions (modern layout: DIM-1, DIM1) ---
declare -A DIMS=(
  [0]="overworld"
  [-1]="nether"
  [1]="end"
)

DIM_JSON="[]"

for DIM_ID in "${!DIMS[@]}"; do
  DIM_NAME="${DIMS[$DIM_ID]}"

  if [[ ${#DIM_FILTER[@]} -gt 0 ]]; then
    skip=true
    for d in "${DIM_FILTER[@]}"; do
      [[ "$DIM_NAME" == "$d" ]] && skip=false
    done
    $skip && continue
  fi

  OUT="${MAP_DIR}/${DIM_NAME}"

  # Determine dimension directory (so we can check for data)
  REGION_DIR="${WORLD_PATH}"
  [[ $DIM_ID -eq -1 ]] && REGION_DIR="${WORLD_PATH}/DIM-1"
  [[ $DIM_ID -eq 1  ]] && REGION_DIR="${WORLD_PATH}/DIM1"

  # Skip missing or empty dimensions
  if [[ ! -d "${REGION_DIR}/region" || -z "$(ls -A "${REGION_DIR}/region" 2>/dev/null)" ]]; then
    echo "⚠️  Skipping ${DIM_NAME} — no region data found in ${REGION_DIR}/region"
    continue
  fi

  echo "→ Rendering ${DIM_NAME} (dimension ${DIM_ID})"
  mkdir -p "$OUT"

  TOPY_ARG=""
  [[ "$DIM_NAME" == "nether" ]] && TOPY_ARG="--topY=120"

  # Render interactive web map
  "${UNMINED}" web render \
    --world="$WORLD_PATH" \
    --dimension="${DIM_ID}" \
    --output="$OUT" \
    --zoomout="$ZOOMOUT" \
    --zoomin="$ZOOMIN" \
    --shadows=3d \
    $TOPY_ARG \
    --players

  # --- Generate preview image ---
  echo "→ Generating preview for ${DIM_NAME}"

  SPAWN_X=$($HOME/.local/bin/nbt -r --path='Data.SpawnX' "$WORLD_PATH/level.dat" 2>/dev/null || echo 0)
  SPAWN_Z=$($HOME/.local/bin/nbt -r --path='Data.SpawnZ' "$WORLD_PATH/level.dat" 2>/dev/null || echo 0)

  RANGE=64
  X1=$((SPAWN_X - RANGE))
  Z1=$((SPAWN_Z - RANGE))
  X2=$((SPAWN_X + RANGE))
  Z2=$((SPAWN_Z + RANGE))
  AREA="b((${X1},${Z1}),(${X2},${Z2}))"

  echo "→ Rendering preview for ${DIM_NAME} around spawn (${SPAWN_X}, ${SPAWN_Z})"
  if ! "${UNMINED}" image render \
      --world="$WORLD_PATH" \
      --dimension="${DIM_ID}" \
      --area="$AREA" \
      --zoom=2 \
      --shadows=3d \
      $TOPY_ARG \
      --trim \
      --output="$OUT/preview.png"; then
    echo "⚠️  Failed to render preview for ${DIM_NAME}, skipping preview image."
  fi

  # Append to JSON manifest
  DIM_JSON=$(echo "$DIM_JSON" | jq --arg name "$DIM_NAME" --argjson id "$DIM_ID" '. + [{"name":$name,"id":$id}]')

  cat >"$OUT/manifest.json" <<EOS
{
  "world": "${WORLD_NAME}",
  "dimension": "${DIM_NAME}",
  "dimension_id": ${DIM_ID},
  "path": "${WORLD_NAME}/${DIM_NAME}",
  "last_rendered": "$(date -Iseconds)"
}
EOS
done

# --- Copy overworld preview to world root ---
if [[ -f "${MAP_DIR}/overworld/preview.png" ]]; then
  cp "${MAP_DIR}/overworld/preview.png" "${MAP_DIR}/preview.png"
fi

# --- Extract metadata from overworld level.dat ---
LEVEL_DAT="${WORLD_BASE}/world/level.dat"
VERSION="unknown"
LASTPLAYED=""

if [[ -f "$LEVEL_DAT" ]]; then
  VERSION=$($HOME/.local/bin/nbt -r --path='Data.Version.Name' "$LEVEL_DAT" 2>/dev/null || echo "unknown")
  LASTPLAYED=$($HOME/.local/bin/nbt -r --path='Data.LastPlayed' "$LEVEL_DAT" 2>/dev/null || echo "")
fi

# Extract difficulty as an integer or null
DIFFICULTY_RAW=$($HOME/.local/bin/nbt -r --path='Data.Difficulty' "$LEVEL_DAT" 2>/dev/null || echo "")
if [[ "$DIFFICULTY_RAW" =~ ^[0-9]+$ ]]; then
  DIFFICULTY_JSON="$DIFFICULTY_RAW"
else
  DIFFICULTY_JSON="null"
fi

DIFF_NAME="Unknown"
case "$DIFFICULTY_RAW" in
  0) DIFF_NAME="Peaceful";;
  1) DIFF_NAME="Easy";;
  2) DIFF_NAME="Normal";;
  3) DIFF_NAME="Hard";;
esac

# convert LastPlayed from ms → seconds → ISO8601 (if present)
if [[ "$LASTPLAYED" =~ ^[0-9]+ ]]; then
  LASTPLAYED_CLEAN="${BASH_REMATCH[0]}"
  LASTPLAYED_ISO=$(date -u -d "@$((LASTPLAYED_CLEAN/1000))" +"%Y-%m-%dT%H:%M:%SZ")
else
  LASTPLAYED_ISO=""
fi

# --- World manifest ---
# --- Generate world manifest (for Lambda/API) ---
jq -n \
  --arg world "$WORLD_NAME" \
  --arg version "$VERSION" \
  --argjson dims "$DIM_JSON" \
  --arg rendered "$(date -Iseconds)" \
  --arg last_played "$LASTPLAYED_ISO" \
  --argjson difficulty "$DIFFICULTY_JSON" \
  --arg difficulty_name "$DIFF_NAME" \
  '{
    world: $world,
    version: $version,
    difficulty: $difficulty,
    difficulty_name: $difficulty_name,
    last_played: ($last_played // ""),
    dimensions: $dims,
    last_rendered: $rendered
  }' > "${MAP_DIR}/manifest.json"

# --- Aggregate world manifests into world_manifest.json ---
AGG_FILE="${MAP_ROOT}/world_manifest.json"

echo "Aggregating world manifests into ${AGG_FILE}"

{
  echo "["
  first=true
  for world_dir in "${MAP_ROOT}"/*/; do
    [[ -d "$world_dir" && -f "${world_dir}/manifest.json" ]] || continue
    if [ "$first" = true ]; then
      first=false
    else
      echo ","
    fi

    world_name=$(basename "$world_dir")
    manifest="${world_dir}/manifest.json"
    preview_rel="${world_name}/preview.png"
    preview_path=""
    if [[ -f "${world_dir}/preview.png" ]]; then
      preview_path="${preview_rel}"
    fi

    # Read manifest content (without outer braces)
    manifest_json=$(jq -c . "$manifest" | sed 's/^{//;s/}$//')

    echo "{${manifest_json},\"preview\":\"${preview_path}\"}"
  done
  echo "]"
} > "${AGG_FILE}"

echo "Generated ${AGG_FILE}"

# --- Rebuild top-level index page ---
INDEX="${MAP_ROOT}/index.html"
ACTIVE_WORLDS=$(systemctl list-units --state=running 'minecraft@*.service' \
    | awk -F'[@.]' '{print $2}')

{
  echo "<!DOCTYPE html><html><head><title>Minecraft Maps</title>"
  echo "<style>
    body{font-family:sans-serif;margin:2rem;background:#f9f9f9;}
    h1{margin-bottom:1rem;}
    h2{margin-top:2rem;margin-bottom:.5rem;}
    small{color:#666;}
    .world{margin-bottom:2rem;padding:1rem;background:#fff;border-radius:12px;box-shadow:0 2px 6px rgba(0,0,0,.1);}
    .preview{max-width:320px;height:auto;border-radius:8px;box-shadow:0 1px 4px rgba(0,0,0,.15);}
    .dim-grid{display:flex;flex-wrap:wrap;gap:1rem;margin-top:.5rem;}
    .dim-card{background:#fafafa;padding:.5rem;border-radius:8px;text-align:center;width:160px;}
    .dim-card img{max-width:150px;height:auto;border-radius:6px;display:block;margin:auto;}
    a{text-decoration:none;color:#0366d6;}
    a:hover{text-decoration:underline;}
  </style>"
  echo "</head><body>"
  echo "<h1>Minecraft Worlds</h1>"

  for world_dir in "${MAP_ROOT}"/*/; do
    [[ -d "$world_dir" ]] || continue
    world_name=$(basename "$world_dir")

    echo "<div class=\"world\">"

    # header + preview
    if echo "$ACTIVE_WORLDS" | grep -qw "$world_name"; then
      echo "<h2>${world_name} <small>(active)</small></h2>"
    else
      echo "<h2>${world_name}</h2>"
    fi

    if [[ -f "${world_dir}/preview.png" ]]; then
      echo "<a href=\"./${world_name}/\"><img class=\"preview\" src=\"./${world_name}/preview.png\" alt=\"${world_name} preview\"></a>"
    fi

    echo "<div class=\"dim-grid\">"
    for dim_dir in "${world_dir}"*/; do
      [[ -d "$dim_dir" ]] || continue
      dim_name=$(basename "$dim_dir")
      [[ "$dim_name" == "overworld" || "$dim_name" == "nether" || "$dim_name" == "end" ]] || continue

      dim_preview="${world_name}/${dim_name}/preview.png"
      echo "<div class=\"dim-card\">"
      if [[ -f "${MAP_ROOT}/${dim_preview}" ]]; then
        echo "<a href=\"./${dim_preview%/preview.png}\"><img src=\"./${dim_preview}\" alt=\"${dim_name}\"></a>"
      fi
      echo "<div><a href=\"./${world_name}/${dim_name}/\">${dim_name}</a></div>"
      echo "</div>"
    done
    echo "</div>"

    echo "</div>"
  done

  echo "</body></html>"
} > "$INDEX"

echo "Landing page regenerated at ${INDEX}"
EOF
sudo chmod 0755 /usr/local/bin/rebuild-map.sh

# Drop-in to hook map rebuild after stop for both units
sudo mkdir -p /etc/systemd/system/minecraft.service.d
sudo tee /etc/systemd/system/minecraft.service.d/override.conf >/dev/null <<'EOF'
[Service]
ExecStopPost=/usr/local/bin/rebuild-map.sh /srv/minecraft-server/world --zoomout=2 --zoomin=1
EOF

sudo mkdir -p /etc/systemd/system/minecraft@.service.d
sudo tee /etc/systemd/system/minecraft@.service.d/override.conf >/dev/null <<'EOF'
[Service]
ExecStopPost=/usr/local/bin/rebuild-map.sh /srv/minecraft-server/%i/world --zoomout=2 --zoomin=1
EOF

sudo systemctl daemon-reexec
sudo systemctl daemon-reload
