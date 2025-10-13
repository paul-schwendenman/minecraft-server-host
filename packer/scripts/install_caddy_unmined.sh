#!/usr/bin/env bash
set -euxo pipefail

MINECRAFT_GROUP="minecraft"
UNMINED_DIR="/opt/unmined"
MAP_DIR="/var/www/map"

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
declare -A DIMS=(
  ["world"]="overworld"
  ["world_nether"]="nether"
  ["world_the_end"]="end"
)
declare -A DIM_IDS=(
  ["overworld"]=0
  ["nether"]=-1
  ["end"]=1
)

for dir in "${!DIMS[@]}"; do
  SRC="${WORLD_BASE}/${dir}"
  DIM_NAME="${DIMS[$dir]}"
  DIM_ID="${DIM_IDS[$DIM_NAME]}"
  OUT="${MAP_DIR}/${DIM_NAME}"

  if [[ -d "$SRC" ]]; then
    echo "→ Rendering ${DIM_NAME} from ${SRC}"
    mkdir -p "$OUT"

    TOPY_ARG=""
    if [[ "$DIM_NAME" == "nether" ]]; then
      TOPY_ARG="--topY=120"
    fi

    "${UNMINED}" web render \
      --world="$SRC" \
      --dimension="${DIM_NAME}" \
      --output="$OUT" \
      --zoomout=6 \
      --zoomin=4 \
      --shadows=3d \
      $TOPY_ARG \
      --players

    # Static preview image
    echo "→ Generating preview for ${DIM_NAME}"
    "${UNMINED}" image render \
      --world="$SRC" \
      --dimension="${DIM_ID}" \
      --area="r((-16,-16),(31,31))" \
      --zoom=-4 \
      --shadows=3d \
      $TOPY_ARG \
      --output="$OUT/preview.png"

    # Dimension manifest
    cat >"$OUT/manifest.json" <<EOS
{
  "world": "${WORLD_NAME}",
  "dimension": "${DIM_NAME}",
  "dimension_id": ${DIM_ID},
  "path": "${WORLD_NAME}/${DIM_NAME}",
  "last_rendered": "$(date -Iseconds)"
}
EOS
  else
    echo "Skipping missing dimension directory: ${SRC}"
  fi
done

# --- Copy overworld preview to world root ---
if [[ -f "${MAP_DIR}/overworld/preview.png" ]]; then
  cp "${MAP_DIR}/overworld/preview.png" "${MAP_DIR}/preview.png"
fi

# --- World manifest ---
jq -n \
  --arg world "$WORLD_NAME" \
  --argjson rendered "$(date -Iseconds)" \
  --argjson dims "$(for d in "${!DIM_IDS[@]}"; do echo "\"$d\""; done | jq -s .)" \
  '{
    world: $world,
    dimensions: $dims,
    last_rendered: now
  }' > "${MAP_DIR}/manifest.json"


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
ExecStopPost=/usr/local/bin/rebuild-map.sh /srv/minecraft-server/world
EOF

sudo mkdir -p /etc/systemd/system/minecraft@.service.d
sudo tee /etc/systemd/system/minecraft@.service.d/override.conf >/dev/null <<'EOF'
[Service]
ExecStopPost=/usr/local/bin/rebuild-map.sh /srv/minecraft-server/%i/world
EOF

sudo systemctl daemon-reexec
sudo systemctl daemon-reload
