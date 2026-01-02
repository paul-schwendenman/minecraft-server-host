#!/usr/bin/env bash
set -euo pipefail

BUCKET="${MC_WORLD_BUCKET:-}"
REGION="${AWS_REGION:-us-east-2}"

if [[ -z "$BUCKET" ]]; then
  echo "No MC_WORLD_BUCKET set; skipping prune"
  exit 0
fi

if [[ -z "${RESTIC_PASSWORD:-}" ]]; then
  echo "No RESTIC_PASSWORD set; skipping prune"
  exit 1
fi

export RESTIC_REPOSITORY="s3:s3.${REGION}.amazonaws.com/${BUCKET}"
export RESTIC_PASSWORD

echo "Pruning old snapshots..."
restic forget --keep-daily 7 --keep-weekly 4 --keep-monthly 3 --prune

echo "Checking repository integrity..."
restic check

echo "Prune complete"
restic stats
