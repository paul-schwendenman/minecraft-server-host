#!/usr/bin/env bash
set -euo pipefail

WORLD="${1:-all}"
WORLD_DIR="/srv/minecraft-server"
BUCKET="${MC_WORLD_BUCKET:-}"
REGION="${AWS_REGION:-us-east-2}"

if [[ -z "$BUCKET" ]]; then
  echo "No MC_WORLD_BUCKET set; skipping"
  exit 0
fi

if [[ -z "${RESTIC_PASSWORD:-}" ]]; then
  echo "No RESTIC_PASSWORD set; skipping"
  exit 1
fi

if ! command -v restic >/dev/null 2>&1; then
  echo "restic not installed; skipping"
  exit 1
fi

export RESTIC_REPOSITORY="s3:s3.${REGION}.amazonaws.com/${BUCKET}"
export RESTIC_PASSWORD

# Initialize repo if needed
if ! restic snapshots --quiet 2>/dev/null; then
  echo "Initializing restic repository..."
  restic init
fi

# Backup
if [[ "$WORLD" == "all" ]]; then
  echo "Backing up all worlds in ${WORLD_DIR}..."
  restic backup "$WORLD_DIR" --tag all \
    --exclude "*.log" --exclude "logs/" --exclude "crash-reports/"
else
  WORLD_PATH="${WORLD_DIR}/${WORLD}/world"
  if [[ ! -d "$WORLD_PATH" ]]; then
    echo "World path not found: $WORLD_PATH"
    exit 1
  fi
  echo "Backing up world: $WORLD..."
  restic backup "$WORLD_PATH" --tag "$WORLD"
fi

echo "Backup complete"
restic snapshots --latest 3 --tag "${WORLD}"
