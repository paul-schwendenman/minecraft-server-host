#!/usr/bin/env bash
set -euo pipefail

# Start the world named by this instance's ActiveWorld tag.
#
# Worlds aren't enabled at boot one by one; this is the only thing that starts
# a world when the instance boots. The control lambda sets the tag before it
# starts the instance. The tag is read from instance metadata, which needs
# instance_metadata_tags enabled on the instance (see infra/modules/mc_stack).
#
# The tag is AWS glue; `minecraftctl world switch` does the actual work: it
# checks the world can start, starts it and waits until it's up, and stops it
# again if it doesn't come up. A missing tag, or a world that can't start or
# doesn't come up, falls back to MC_DEFAULT_WORLD (from /etc/minecraft.env,
# default "default"), so a bad tag can't leave the server up with no world
# running.

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
elif [[ "${WORLD}" != "${DEFAULT_WORLD}" ]]; then
  log "Starting ActiveWorld ${WORLD}"
  if minecraftctl world switch "${WORLD}"; then
    exit 0
  fi
  log "ActiveWorld '${WORLD}' didn't start, using ${DEFAULT_WORLD}"
fi

log "Starting ${DEFAULT_WORLD}"
if ! minecraftctl world switch "${DEFAULT_WORLD}"; then
  log "Default world '${DEFAULT_WORLD}' didn't start, nothing is running"
  exit 1
fi
