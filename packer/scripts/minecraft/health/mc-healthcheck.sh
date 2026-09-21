#!/usr/bin/env bash
set -euo pipefail

echo "=== Minecraft Server Health Check ==="

# 1. World services ------------------------------------------------------------
echo "[*] Checking Minecraft services..."
ACTIVE_UNITS=$(systemctl list-units --type=service --state=running 'minecraft@*.service' \
    --no-legend | awk -F'[@.]' '{print $2}' || true)

if [[ -z "$ACTIVE_UNITS" ]]; then
    echo "  ✘ No running minecraft@*.service units"
else
    for WORLD in $ACTIVE_UNITS; do
        if systemctl is-active --quiet "minecraft@$WORLD.service"; then
            echo "  ✔ minecraft@$WORLD.service running"
        else
            echo "  ✘ minecraft@$WORLD.service NOT running"
        fi
    done
fi

# 2. Caddy --------------------------------------------------------------------
echo "[*] Checking Caddy..."
if systemctl is-active --quiet caddy; then
    echo "  ✔ Caddy is active"
else
    echo "  ✘ Caddy inactive"
fi

# 3. Autoshutdown timer --------------------------------------------------------
echo "[*] Checking autoshutdown..."
if systemctl is-enabled --quiet autoshutdown.timer; then
    echo "  ✔ autoshutdown.timer enabled"
else
    echo "  ✘ autoshutdown.timer not enabled"
fi
if systemctl is-failed --quiet autoshutdown.service; then
    echo "  ✘ autoshutdown.service last run FAILED (journalctl -u autoshutdown.service)"
else
    echo "  ✔ autoshutdown.service last run ok"
fi

# 4. Dynamic DNS service -------------------------------------------------------
echo "[*] Checking dyndns..."
if systemctl is-enabled --quiet dyndns.service 2>/dev/null; then
    if systemctl is-active --quiet dyndns.service; then
        echo "  ✔ dyndns.service ran successfully"
    else
        echo "  ✘ dyndns.service failed"
    fi
else
    echo "  ⚠️ dyndns.service not enabled (Route53 DNS updates disabled)"
fi

# 5. Map build timers ---------------------------------------------------------
echo "[*] Checking map build timers..."
BUILD_TIMERS=$(systemctl list-timers --all --no-legend | grep minecraft-map-build || true)
if [[ -n "$BUILD_TIMERS" ]]; then
    echo "$BUILD_TIMERS" | while read -r line; do echo "  $line"; done
else
    echo "  (no map build timers active)"
fi

# 6. Symlink + directories -----------------------------------------------------
echo "[*] Checking map symlink..."
if [[ -L /var/www/map ]]; then
    echo "  ✔ /var/www/map → $(readlink -f /var/www/map)"
else
    echo "  ✘ /var/www/map missing or not symlink"
fi

# 7. Listening ports -----------------------------------------------------------
echo "[*] Checking expected ports..."
ss -tulwn | grep -E '(:25565|:80|:443)' || echo "  ✘ Expected ports not open"

# 8. RCON query ---------------------------------------------------------------
echo "[*] Querying RCON..."
if [ ! -e /etc/minecraft.env ]; then
  echo "  ✘ /etc/minecraft.env missing"
elif [ ! -r /etc/minecraft.env ]; then
  echo "  ⚠️ /etc/minecraft.env not readable (permissions)"
else
    source /etc/minecraft.env
    export RCON_PASSWORD RCON_PORT

    if command -v minecraftctl >/dev/null 2>&1; then
        if ! minecraftctl rcon send "list" >/dev/null 2>&1; then
            echo "  ✘ RCON query failed"
        else
            echo "  ✔ RCON responsive"
        fi
    else
        echo "  (minecraftctl not installed)"
    fi
fi

# 9. Backups -------------------------------------------------------------
echo "[*] Checking backup timers..."
MAP_BACKUP_TIMERS=$(systemctl list-timers --all --no-legend | grep minecraft-map-backup || true)
if [[ -n "$MAP_BACKUP_TIMERS" ]]; then
    echo "$MAP_BACKUP_TIMERS" | while read -r line; do echo "  $line"; done
else
    echo "  ⚠️ No map backup timers active"
fi

WORLD_BACKUP_TIMERS=$(systemctl list-timers --all --no-legend | grep minecraft-world-backup || true)
if [[ -n "$WORLD_BACKUP_TIMERS" ]]; then
    echo "$WORLD_BACKUP_TIMERS" | while read -r line; do echo "  $line"; done
else
    echo "  ⚠️ No world backup timers active"
fi

WORLD_PRUNE_TIMERS=$(systemctl list-timers --all --no-legend | grep minecraft-world-prune || true)
if [[ -n "$WORLD_PRUNE_TIMERS" ]]; then
    echo "$WORLD_PRUNE_TIMERS" | while read -r line; do echo "  $line"; done
else
    echo "  ⚠️ No world prune timer active"
fi

# 10. EBS Mount -----------------------------------------------------------
echo "[*] Checking EBS mount (/srv/minecraft-server)..."
if mountpoint -q /srv/minecraft-server; then
  echo "  ✔ Mounted"
else
  echo "  ✘ Not mounted"
fi
if [[ -w /srv/minecraft-server ]]; then
  echo "  ✔ /srv/minecraft-server writable by $(id -un)"
else
  echo "  ✘ /srv/minecraft-server NOT writable by $(id -un) (owner: $(stat -c '%u:%g' /srv/minecraft-server))"
fi

# 11. CLI Tools ----------------------------------------------------------
for bin in minecraftctl aws java caddy restic; do
  if command -v "$bin" >/dev/null 2>&1; then
    echo "  ✔ $bin installed"
  else
    echo "  ✘ $bin missing"
  fi
done
[[ -x /opt/unmined/unmined-cli ]] && echo "  ✔ uNmINeD installed" || echo "  ✘ uNmINeD missing"

# 12. Disk usage ---------------------------------------------------------
echo "[*] Checking disk usage..."
df -h /srv/minecraft-server | awk 'NR==2 {print "  📦 "$5 " used on "$6}'

# 13. Failed units -------------------------------------------------------
echo "[*] Checking for failed Minecraft units..."
systemctl --failed | grep minecraft || echo "  ✔ No failed units"

echo "=== Health check complete ==="
