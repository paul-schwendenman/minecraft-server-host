#!/usr/bin/env bash
set -euo pipefail

WORLD="${1:-all}"
BASE_DIR="/var/www/map"
BUCKET="${MC_MAP_BUCKET:-}"

if [[ -z "$BUCKET" ]]; then
  echo "No MC_MAP_BUCKET set; skipping map backup"
  exit 0
fi

if ! command -v aws >/dev/null 2>&1; then
  echo "AWS CLI not installed; skipping map backup"
  exit 0
fi

backup_world() {
  local world="$1"
  local dir="${BASE_DIR}/${world}"

  if [[ -d "$dir" ]]; then
    echo "Backing up maps for $world → s3://$BUCKET/maps/$world/"


    ### Replace aws sync
    # rclone sync /srv/minecraft-server/maps/ s3:minecraft-test-maps/maps \
    #   --s3-provider AWS \
    #   --s3-region us-east-2 \
    #   --fast-list \
    #   --checkers 16 \
    #   --transfers 8 \
    #   --progress

    aws s3 sync "$dir" "s3://$BUCKET/maps/$world/" --delete
    echo "Backup complete for $world."
  else
    echo "Skipping missing map directory: $dir"
  fi
}

if [[ "$WORLD" == "all" ]]; then
  echo "Backing up all maps under $BASE_DIR"
  for d in "${BASE_DIR}"/*; do
    [[ -d "$d" ]] && backup_world "$(basename "$d")" || true
  done
  echo "All map backups complete."
else
  backup_world "$WORLD"
fi
