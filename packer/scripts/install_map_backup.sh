#!/usr/bin/env bash
set -euxo pipefail

sudo tee /usr/local/bin/backup-maps.sh >/dev/null <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

WORLD="${1:-world}"
MAP_DIR="/var/www/map/${WORLD}"
BUCKET="${MC_MAP_BUCKET:-}"

if [[ -z "$BUCKET" ]]; then
  echo "No MC_MAP_BUCKET set; skipping map backup"
  exit 0
fi

if ! command -v aws >/dev/null 2>&1; then
  echo "AWS CLI not installed; skipping map backup"
  exit 0
fi

if [[ -d "$MAP_DIR" ]]; then
  echo "Backing up maps for $WORLD to s3://$BUCKET/maps/$WORLD/"

### Replace aws sync
# rclone sync /srv/minecraft-server/maps/ s3:minecraft-test-maps/maps \
#   --s3-provider AWS \
#   --s3-region us-east-2 \
#   --fast-list \
#   --checkers 16 \
#   --transfers 8 \
#   --progress

#   --s3-acl public-read \

  aws s3 sync "$MAP_DIR" "s3://$BUCKET/maps/$WORLD/" --delete
fi
EOF

sudo chmod 0755 /usr/local/bin/backup-maps.sh
