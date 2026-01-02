#!/usr/bin/env bash
set -euo pipefail

BUCKET="${MC_WORLD_BUCKET:-}"
REGION="${AWS_REGION:-us-east-2}"

if [[ -z "$BUCKET" ]]; then
  echo "Error: MC_WORLD_BUCKET not set"
  exit 1
fi

if [[ -z "${RESTIC_PASSWORD:-}" ]]; then
  echo "Error: RESTIC_PASSWORD not set"
  exit 1
fi

export RESTIC_REPOSITORY="s3:s3.${REGION}.amazonaws.com/${BUCKET}"
export RESTIC_PASSWORD

usage() {
  echo "Usage: $0 <command> [options]"
  echo ""
  echo "Commands:"
  echo "  list [tag]                  List available snapshots (optionally filter by tag)"
  echo "  restore <snapshot>          Restore a specific snapshot to original location"
  echo "  restore-to <snapshot> <path> Restore snapshot to custom path"
  echo ""
  echo "Examples:"
  echo "  $0 list"
  echo "  $0 list survival"
  echo "  $0 restore latest"
  echo "  $0 restore abc123"
  echo "  $0 restore-to abc123 /tmp/restore-test"
  exit 1
}

case "${1:-}" in
  list)
    if [[ -n "${2:-}" ]]; then
      restic snapshots --tag "$2"
    else
      restic snapshots
    fi
    ;;
  restore)
    SNAPSHOT="${2:-}"
    if [[ -z "$SNAPSHOT" ]]; then
      echo "Error: snapshot ID required"
      usage
    fi
    echo "Restoring snapshot $SNAPSHOT to original location..."
    restic restore "$SNAPSHOT" --target /
    echo "Restore complete"
    ;;
  restore-to)
    SNAPSHOT="${2:-}"
    TARGET="${3:-}"
    if [[ -z "$SNAPSHOT" ]] || [[ -z "$TARGET" ]]; then
      echo "Error: snapshot ID and target path required"
      usage
    fi
    echo "Restoring snapshot $SNAPSHOT to $TARGET..."
    mkdir -p "$TARGET"
    restic restore "$SNAPSHOT" --target "$TARGET"
    echo "Restore complete at $TARGET"
    ;;
  *)
    usage
    ;;
esac
