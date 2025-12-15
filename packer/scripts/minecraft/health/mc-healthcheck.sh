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

# 4. Map rebuild timers --------------------------------------------------------
echo "[*] Checking map rebuild timers..."
if systemctl list-unit-files | grep -q minecraft-map-rebuild@; then
    systemctl list-timers --all | grep minecraft-map-rebuild || echo "  (no active timers)"
else
    echo "  ✘ minecraft-map-rebuild units not installed"
fi

# 5. Symlink + directories -----------------------------------------------------
echo "[*] Checking map symlink..."
if [[ -L /var/www/map ]]; then
    echo "  ✔ /var/www/map → $(readlink -f /var/www/map)"
else
    echo "  ✘ /var/www/map missing or not symlink"
fi

# 6. Listening ports -----------------------------------------------------------
echo "[*] Checking expected ports..."
ss -tulwn | grep -E '(:25565|:80|:443)' || echo "  ✘ Expected ports not open"

# 7. RCON query ---------------------------------------------------------------
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

# 8. Backups -------------------------------------------------------------
echo "[*] Checking backup timers..."
for unit in minecraft-map-backup.timer minecraft-world-backup.timer; do
  if systemctl list-unit-files | grep -q "$unit"; then
    if systemctl is-enabled --quiet "$unit"; then
      echo "  ✔ $unit enabled"
    else
      echo "  ⚠️ $unit present but not enabled"
    fi
  else
    echo "  ✘ $unit missing"
  fi
done

# 9. EBS Mount -----------------------------------------------------------
echo "[*] Checking EBS mount (/srv/minecraft-server)..."
if mountpoint -q /srv/minecraft-server; then
  echo "  ✔ Mounted"
else
  echo "  ✘ Not mounted"
fi

# 10. CLI Tools ----------------------------------------------------------
for bin in minecraftctl aws java caddy; do
  if command -v "$bin" >/dev/null 2>&1; then
    echo "  ✔ $bin installed"
  else
    echo "  ✘ $bin missing"
  fi
done
[[ -x /opt/unmined/unmined-cli ]] && echo "  ✔ uNmINeD installed" || echo "  ✘ uNmINeD missing"

# 11. Disk usage ---------------------------------------------------------
echo "[*] Checking disk usage..."
df -h /srv/minecraft-server | awk 'NR==2 {print "  📦 "$5 " used on "$6}'

# 12. Failed units -------------------------------------------------------
echo "[*] Checking for failed Minecraft units..."
systemctl --failed | grep minecraft || echo "  ✔ No failed units"

echo "=== Health check complete ==="
