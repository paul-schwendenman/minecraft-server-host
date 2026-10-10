#!/usr/bin/env bash
set -euo pipefail

# Run an ExecStopPost command only when the stop is worth acting on.
#
# Usage: minecraft-stop-gate --unit UNIT [--min-uptime SECS] -- CMD...
#
# systemd passes SERVICE_RESULT, EXIT_CODE and EXIT_STATUS to ExecStopPost.
# CMD runs when SERVICE_RESULT is "success" (or unset, i.e. not run by
# systemd). Otherwise, with --min-uptime, CMD still runs if the unit's main
# process ran for at least SECS, read from systemd's own ExecMain*Timestamp
# properties. Anything else is skipped with exit 0, so a crash-looping world
# doesn't run a backup or map build on every failed start.
#
# If the uptime can't be worked out, CMD runs: a pointless backup is cheaper
# than a lost one.
#
# Messages go to stderr, which systemd logs under the unit.

usage() {
  echo "Usage: minecraft-stop-gate --unit UNIT [--min-uptime SECS] -- CMD..." >&2
  exit 2
}

log() {
  echo "minecraft-stop-gate: $1" >&2
}

is_number() {
  [[ "$1" =~ ^[0-9]+$ ]]
}

UNIT=""
MIN_UPTIME=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --unit)
      [[ $# -ge 2 ]] || usage
      UNIT="$2"
      shift 2
      ;;
    --min-uptime)
      if [[ $# -lt 2 ]] || ! is_number "$2"; then usage; fi
      MIN_UPTIME="$2"
      shift 2
      ;;
    --)
      shift
      break
      ;;
    *) usage ;;
  esac
done

# The loop only ends with $# == 0 if `--` never came, or nothing followed it
[[ -n "${UNIT}" && $# -gt 0 ]] || usage

RESULT="${SERVICE_RESULT:-success}"

if [[ "${RESULT}" == "success" ]]; then
  exec "$@"
fi

UPTIME="unknown"
if [[ -n "${MIN_UPTIME}" ]]; then
  START=$(systemctl show -P ExecMainStartTimestampMonotonic "${UNIT}" 2>/dev/null || true)
  EXIT=$(systemctl show -P ExecMainExitTimestampMonotonic "${UNIT}" 2>/dev/null || true)

  if ! is_number "${START}" || ! is_number "${EXIT}" \
    || [[ "${START}" -eq 0 || "${EXIT}" -le "${START}" ]]; then
    log "${UNIT} uptime unknown, running $1 (SERVICE_RESULT=${RESULT})"
    exec "$@"
  fi

  UPTIME=$(((EXIT - START) / 1000000))
  if [[ "${UPTIME}" -ge "${MIN_UPTIME}" ]]; then
    exec "$@"
  fi
  UPTIME="${UPTIME}s"
fi

log "skipping $1: SERVICE_RESULT=${RESULT} EXIT_CODE=${EXIT_CODE:-} EXIT_STATUS=${EXIT_STATUS:-} uptime=${UPTIME}"
exit 0
