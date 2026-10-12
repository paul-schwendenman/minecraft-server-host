#!/usr/bin/env bash
set -Eeuo pipefail

# Idle-check state lives on tmpfs (/run), created for the service user by
# RuntimeDirectory= in autoshutdown.service. It must NOT live on the EBS volume:
# that volume's root directory keeps the numeric owner from whichever AMI created
# it, so the minecraft user can silently lose write access after an AMI rebuild.
# tmpfs also means a stale marker never survives a reboot.
STATE_DIR="/run/autoshutdown"
TOUCH_FILE="${STATE_DIR}/no_one_playing"

# Unexpected failures under `set -e` used to be visible only as a failed unit.
trap 'logger -p user.err -t autoshutdown "Unexpected failure at line ${LINENO} (exit status $?)"' ERR

power_off() {
  logger -t autoshutdown "$1, shutting down now"
  rm -f "${TOUCH_FILE}"
  sudo /sbin/poweroff
  exit 0
}

# Skip shutdown while someone has an interactive SSH session. sshd titles each
# session's process "<user>@pts/N" (or "<user>@notty" for `ssh host cmd`), as
# "sshd-session: ..." on OpenSSH >= 9.8 and "sshd: ..." before. `who` can't be
# used: Ubuntu 26.04 has no /run/utmp, so it never lists anyone.
SSH_SESSIONS=$(pgrep -af '^sshd(-session)?: [^ ]+@pts/' || true)
if [[ -n "${SSH_SESSIONS}" ]]; then
  rm -f "${TOUCH_FILE}"
  logger -t autoshutdown "Skipping shutdown: active SSH session(s): $(echo "${SSH_SESSIONS}" | sed 's/^[0-9]* sshd[^:]*: //' | paste -sd ' ' -)"
  exit 0
fi

# Skip while `minecraftctl world switch` runs (it holds this lock): mid-switch
# no world may be running, which would otherwise power off straight away. Keep
# holding it until this script exits, so a switch can't start between the check
# and the shutdown decision below (a switch waits a few seconds for it).
# minecraft-switch-lock.conf (tmpfiles.d) creates the file at boot; this user
# can't create files in /run, so without it there's nothing to check.
SWITCH_LOCK="/run/minecraft-switch.lock"
if [[ -e "${SWITCH_LOCK}" ]]; then
  exec 9<"${SWITCH_LOCK}"
  if ! flock -n 9; then
    rm -f "${TOUCH_FILE}"
    logger -t autoshutdown "Skipping shutdown: world switch in progress"
    exit 0
  fi
else
  logger -p user.warning -t autoshutdown "${SWITCH_LOCK} missing, can't tell whether a world switch is running"
fi

# Check if any minecraft service is running
RUNNING_SERVICES=$(systemctl list-units --type=service --state=running 'minecraft@*.service' --no-legend 2>/dev/null | wc -l)

if [[ "${RUNNING_SERVICES}" -eq 0 ]]; then
  power_off "No minecraft services running"
fi

# Load shared RCON settings for minecraftctl
# minecraftctl reads RCON_PASSWORD, RCON_HOST, and RCON_PORT from environment variables
if [[ -r /etc/minecraft.env ]]; then
  source /etc/minecraft.env
else
  echo "Error: /etc/minecraft.env not readable" >&2
  exit 1
fi

command -v minecraftctl >/dev/null 2>&1 || {
  echo "Error: minecraftctl not found" >&2
  exit 1
}

# Capture stderr too: minecraftctl logs why RCON failed there. Bound the call so
# a wedged server can't hang the check (the unit also has TimeoutStartSec=).
# Any failure counts as 0 players, so a broken RCON still lets the box shut down.
RCON_STATUS=0
OUTPUT=$(timeout 30 minecraftctl rcon send "list" 2>&1) || RCON_STATUS=$?
COUNT=$(echo "$OUTPUT" | sed -n 's/.*There are \([0-9]*\).*/\1/p' | head -1)
if [[ -z "${COUNT}" ]]; then
  if [[ "${RCON_STATUS}" -eq 124 ]]; then
    logger -p user.warning -t autoshutdown "RCON timed out after 30s, treating as 0 players"
  elif [[ "${OUTPUT}" == *"authentication failed"* ]]; then
    logger -p user.warning -t autoshutdown "RCON auth failed (password mismatch; try 'minecraftctl config sync <world>'), treating as 0 players"
  else
    logger -t autoshutdown "Failed to parse player count from RCON output (exit ${RCON_STATUS}): ${OUTPUT}"
  fi
  COUNT=0
fi
logger -t autoshutdown "RCON reports $COUNT players online"

if [[ "${COUNT}" -eq 0 ]]; then
  if [[ -f "${TOUCH_FILE}" ]]; then
    power_off "No players - second check"
  fi

  if mkdir -p "${STATE_DIR}" && touch "${TOUCH_FILE}"; then
    logger -t autoshutdown "No players — first check, will shut down next time"
  else
    # Without the marker there is no way to count consecutive idle checks. Fail
    # toward shutdown (one idle check instead of two) rather than running forever.
    logger -p user.err -t autoshutdown "Cannot write ${TOUCH_FILE}; treating this idle check as the second"
    power_off "No players - state file unwritable"
  fi
else
  rm -f "${TOUCH_FILE}"
  logger -t autoshutdown "Players online — skipping shutdown"
fi
