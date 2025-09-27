#!/usr/bin/env bash
set -euxo pipefail

sudo tee /usr/local/bin/backup-worlds.sh >/dev/null <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

WORLD_DIR="/srv/minecraft-server"
BUCKET="${MC_WORLD_BUCKET:-}"

if [[ -z "$BUCKET" ]]; then
  echo "No MC_WORLD_BUCKET set; skipping world backup"
  exit 0
fi

if ! command -v aws >/dev/null 2>&1; then
  echo "AWS CLI not installed; skipping world backup"
  exit 0
fi

echo "Creating tarball of all worlds"
TMPFILE=$(mktemp /tmp/worlds-XXXXXX.tar.gz)
tar -czf "$TMPFILE" -C "$WORLD_DIR" .

echo "Uploading worlds to s3://$BUCKET/worlds/"
aws s3 cp "$TMPFILE" "s3://$BUCKET/worlds/worlds-\$(date +%F-%H%M).tar.gz"

rm -f "$TMPFILE"
EOF

sudo chmod 0755 /usr/local/bin/backup-worlds.sh
