#!/usr/bin/env bash
set -euo pipefail

# Start the world named by this instance's ActiveWorld tag.
#
# Worlds aren't enabled at boot one by one; this is the only thing that starts
# a world when the instance boots. The control lambda sets the tag before it
# starts the instance. The tag is read from instance metadata, which needs
# instance_metadata_tags enabled on the instance (see infra/modules/mc_stack).
#
# A missing tag, or a world that can't start, falls back to MC_DEFAULT_WORLD
# (from /etc/minecraft.env, default "default"), so a bad tag can't leave the
# server up with no world running.

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

# Why a world can't start, or nothing if it can. Doesn't require
# world/level.dat: Minecraft writes that on a world's first start, so a world
# fresh from `minecraftctl world create` has none yet.
why_not_startable() {
  local dir="${WORLDS_DIR}/$1"
  if [[ ! "$1" =~ ^[A-Za-z0-9][A-Za-z0-9._-]*$ ]]; then
    echo "not a valid world name"
  elif [[ ! -f "${dir}/server.properties" ]]; then
    echo "no ${dir}/server.properties"
  elif [[ ! -f "${dir}/server.jar" ]]; then
    # -f follows the symlink, so this also catches a jar missing from
    # /opt/minecraft/jars
    echo "${dir}/server.jar is missing or points to a missing jar"
  fi
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
else
  REASON=$(why_not_startable "${WORLD}")
  if [[ -n "${REASON}" ]]; then
    log "ActiveWorld '${WORLD}' can't start (${REASON}), using ${DEFAULT_WORLD}"
    WORLD="${DEFAULT_WORLD}"
  fi
fi

REASON=$(why_not_startable "${WORLD}")
if [[ -n "${REASON}" ]]; then
  log "Default world '${WORLD}' can't start (${REASON}), not starting anything"
  exit 1
fi

log "Starting minecraft@${WORLD}.service"
systemctl start "minecraft@${WORLD}.service"
