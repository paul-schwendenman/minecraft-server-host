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
  ["world_the_end"]="the_end"
)

for dir in "${!DIMS[@]}"; do
  SRC="${WORLD_BASE}/${dir}"
  DIM="${DIMS[$dir]}"
  OUT="${MAP_DIR}/${DIM}"

  if [[ -d "$SRC" ]]; then
    echo "→ Rendering ${DIM} from ${SRC}"
    mkdir -p "$OUT"
    "${UNMINED}" web render \
      --world="$SRC" \
      --output="$OUT" \
      --zoomout=6 \
      --zoomin=4 \
      --shadows=3d \
      --players

    # preview + manifest for S3 / UI
    #if [[ -f "$OUT/screenshot.png" ]]; then
    #  convert "$OUT/screenshot.png" -resize 320x180 "$OUT/preview.png" || true
    #fi
    if [[ -f "$OUT/preview.png" ]]; then
      "${UNMINED}" image render \
        --world="$SRC" \
        --area="r((-16,-16),(31,31))" \
        --zoom=-4 \
        --dimension=0 \
        --shadows=3d \
        --output="$OUT/preview.png"
    fi

    cat >"$OUT/manifest.json" <<EOS
{
  "world": "${WORLD_NAME}",
  "dimension": "${DIM}",
  "last_rendered": "$(date -Iseconds)"
}
EOS
  else
    echo "Skipping missing dimension directory: ${SRC}"
  fi
done

# --- Rebuild landing page (same as before) ---
INDEX="${MAP_ROOT}/index.html"
ACTIVE_WORLDS=$(systemctl list-units --state=running 'minecraft@*.service' \
    | awk -F'[@.]' '{print $2}')

{
  echo "<html><head><title>Minecraft Worlds</title></head><body>"
  echo "<h1>Available Worlds</h1><ul>"
  for d in "${MAP_ROOT}"/*/; do
    name=$(basename "$d")
    if echo "$ACTIVE_WORLDS" | grep -qw "$name"; then
      echo "<li><b><a href=\"./${name}/\">${name}</a> (active)</b></li>"
    else
      echo "<li><a href=\"./${name}/\">${name}</a></li>"
    fi
  done
  echo "</ul></body></html>"
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
