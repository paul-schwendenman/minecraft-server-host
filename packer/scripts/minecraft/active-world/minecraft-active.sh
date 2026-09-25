#!/usr/bin/env bash
set -euo pipefail

# Start the world named by this instance's ActiveWorld tag.
#
# Worlds aren't enabled at boot one by one; this is the only thing that starts
# a world when the instance boots. The control lambda sets the tag before it
# starts the instance. The tag is read from instance metadata, which needs
# instance_metadata_tags enabled on the instance (see infra/modules/mc_stack).
#
# A missing tag, an invalid name or a world that doesn't exist falls back to
# MC_DEFAULT_WORLD (from /etc/minecraft.env, default "default"), so a bad tag
# can't leave the server up with no world running.

WORLDS_DIR="/srv/minecraft-server"
IMDS="http://169.254.169.254/latest"

log() {
  echo "$1"
  logger -t minecraft-active "$1" || true
}

if [[ -r /etc/minecraft.env ]]; then
  # shellcheck disable=SC1091
  source /etc/minecraft.env
fi
DEFAULT_WORLD="${MC_DEFAULT_WORLD:-default}"

world_exists() {
  [[ "$1" =~ ^[A-Za-z0-9][A-Za-z0-9._-]*$ ]] && [[ -f "${WORLDS_DIR}/$1/world/level.dat" ]]
}

# IMDSv2. A missing tag (or tags not exposed to metadata) is a 404, so curl -f
# leaves WORLD empty.
WORLD=""
TOKEN=$(curl -sf -X PUT "${IMDS}/api/token" \
  -H "X-aws-ec2-metadata-token-ttl-seconds: 60" || true)
if [[ -n "${TOKEN}" ]]; then
  WORLD=$(curl -sf -H "X-aws-ec2-metadata-token: ${TOKEN}" \
    "${IMDS}/meta-data/tags/instance/ActiveWorld" || true)
else
  log "Could not get an instance metadata token"
fi

if [[ -z "${WORLD}" ]]; then
  log "No ActiveWorld tag, using ${DEFAULT_WORLD}"
  WORLD="${DEFAULT_WORLD}"
elif ! world_exists "${WORLD}"; then
  log "ActiveWorld '${WORLD}' is not a world in ${WORLDS_DIR}, using ${DEFAULT_WORLD}"
  WORLD="${DEFAULT_WORLD}"
fi

if ! world_exists "${WORLD}"; then
  log "Default world '${WORLD}' not found in ${WORLDS_DIR}, not starting anything"
  exit 1
fi

log "Starting minecraft@${WORLD}.service"
systemctl start "minecraft@${WORLD}.service"
