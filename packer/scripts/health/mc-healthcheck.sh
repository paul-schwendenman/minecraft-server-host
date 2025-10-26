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
if [[ -r /etc/minecraft.env ]]; then
    source /etc/minecraft.env
    if command -v mcrcon >/dev/null 2>&1; then
        echo "[*] Querying RCON..."
        if ! echo "list" | mcrcon -H 127.0.0.1 -P "${RCON_PORT:-25575}" -p "${RCON_PASSWORD:-}" >/dev/null; then
            echo "  ✘ RCON query failed"
        else
            echo "  ✔ RCON responsive"
        fi
    else
        echo "  (mcrcon not installed)"
    fi
else
    echo "  (no /etc/minecraft.env found, skipping RCON)"
fi

echo "=== Health check complete ==="
