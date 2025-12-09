#!/usr/bin/env bash
set -euo pipefail

WORLD="${1:-all}"
WORLD_DIR="/srv/minecraft-server"
BUCKET="${MC_WORLD_BUCKET:-}"
TMPFILE=$(mktemp /tmp/worlds-XXXXXX.tar.gz)

if [[ -z "$BUCKET" ]]; then
  echo "No MC_WORLD_BUCKET set; skipping world backup"
  exit 0
fi

if ! command -v aws >/dev/null 2>&1; then
  echo "AWS CLI not installed; skipping world backup"
  exit 0
fi

if [[ "$WORLD" == "all" ]]; then
  echo "Creating tarball of all worlds..."
  tar -czf "$TMPFILE" -C "$WORLD_DIR" .
else
  WORLD_PATH="${WORLD_DIR}/${WORLD}/world"
  if [[ ! -d "$WORLD_PATH" ]]; then
    echo "World path not found: $WORLD_PATH"
    exit 1
  fi
  echo "Creating tarball for $WORLD..."
  tar -czf "$TMPFILE" -C "$WORLD_PATH" .
fi

DEST="s3://$BUCKET/worlds/${WORLD}-$(date +%F-%H%M).tar.gz"
echo "Uploading backup to $DEST"
aws s3 cp "$TMPFILE" "$DEST"

rm -f "$TMPFILE"
echo "✅ Backup complete for $WORLD"
