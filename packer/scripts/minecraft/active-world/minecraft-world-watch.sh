#!/usr/bin/env bash
set -euo pipefail

# Switch worlds when this instance's ActiveWorld tag changes while it runs.
#
# The control lambda sets the tag (POST /switch?world=<name> on a running
# server); this runs every minute from minecraft-world-watch.timer and calls
# `minecraftctl world switch`, which does the work. Like minecraft-active.sh,
# it's AWS glue: minecraftctl knows nothing about tags.
#
# It acts on tag *changes*, not on the tag differing from the running world:
# HANDLED_FILE holds the last tag value acted on (minecraft-active.sh writes it
# at boot), so a switch done by hand over SSH stays until the tag changes again.
# What happens to a changed tag depends on `world switch`'s exit code:
#
#   0     switched (or already running it)         -> handled
#   2, 4  can't start / didn't come up, rolled back -> handled (not retried)
#   5     didn't come up, nor did the previous one  -> handled, logged as error
#   3     refused: players online, or busy          -> retried next run
#   other unexpected error                          -> retried next run

IMDS="http://169.254.169.254/latest"
HANDLED_FILE="/run/minecraft-world-tag"

# log [priority] message
log() {
  local priority="user.notice"
  if [[ $# -gt 1 ]]; then
    priority="$1"
    shift
  fi
  echo "$1"
  logger -p "${priority}" -t minecraft-world-watch "$1" || true
}

TOKEN=$(curl -sf -X PUT "${IMDS}/api/token" \
  -H "X-aws-ec2-metadata-token-ttl-seconds: 60" || true)
if [[ -z "${TOKEN}" ]]; then
  log "Could not get an instance metadata token"
  exit 0
fi
# A missing tag is a 404, which leaves TAG empty
TAG=$(curl -sf -H "X-aws-ec2-metadata-token: ${TOKEN}" \
  "${IMDS}/meta-data/tags/instance/ActiveWorld" || true)

if [[ ! -e "${HANDLED_FILE}" ]]; then
  # minecraft-active.sh didn't record what it started; take the current tag
  # as the starting point rather than guessing.
  log "No ${HANDLED_FILE}; treating ActiveWorld '${TAG}' as handled"
  printf '%s' "${TAG}" > "${HANDLED_FILE}"
  exit 0
fi

HANDLED=$(cat "${HANDLED_FILE}")
if [[ "${TAG}" == "${HANDLED}" ]]; then
  exit 0
fi
if [[ -z "${TAG}" ]]; then
  # Removing the tag means "the default world at next boot"; don't switch now
  log "ActiveWorld tag removed; leaving the running world alone"
  printf '%s' "${TAG}" > "${HANDLED_FILE}"
  exit 0
fi

log "ActiveWorld changed from '${HANDLED}' to '${TAG}', switching"
STATUS=0
minecraftctl world switch "${TAG}" || STATUS=$?

case "${STATUS}" in
  0)
    log "Switched to ${TAG}"
    ;;
  2 | 4)
    log user.warning "Didn't switch to ${TAG} (exit ${STATUS}); not retrying until the tag changes"
    ;;
  5)
    log user.err "${TAG} didn't start and neither did the previous world; nothing is running"
    ;;
  3)
    log "Switch to ${TAG} refused for now (players online or another switch running); will retry"
    exit 0
    ;;
  *)
    log user.err "Switch to ${TAG} failed (exit ${STATUS}); will retry"
    exit 0
    ;;
esac
printf '%s' "${TAG}" > "${HANDLED_FILE}"
