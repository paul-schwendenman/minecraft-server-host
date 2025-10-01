#!/usr/bin/env bash
set -euxo pipefail

sudo tee /usr/local/bin/mc-healthcheck.sh >/dev/null <<'EOF'
#!/bin/bash
set -euo pipefail

echo "=== Minecraft Server Health Check ==="

# 1. Check Minecraft systemd service(s)
echo "[*] Checking Default Minecraft server..."
if systemctl is-active --quiet "minecraft@default.service"; then
    echo "  ✔ minecraft@default.service is running"
else
    echo "  ✘ minecraft@default.service is NOT running"
    systemctl status minecraft@default.service --no-pager
fi

echo "[*] Checking Minecraft world services..."
ACTIVE_WORLDS=$(systemctl list-units --state=running 'minecraft@*.service' \
    --no-legend | awk -F'[@.]' '{print $2}' || true)

if [[ -z "$ACTIVE_WORLDS" ]]; then
    echo "  ✘ No active minecraft@*.service units found"
else
    for WORLD in $ACTIVE_WORLDS; do
        if systemctl is-active --quiet "minecraft@$WORLD.service"; then
            echo "  ✔ minecraft@$WORLD.service is running"
        else
            echo "  ✘ minecraft@$WORLD.service is NOT running"
            systemctl status "minecraft@$WORLD.service" --no-pager
        fi
    done
fi

# 2. Check Caddy
echo "[*] Checking Caddy..."
if systemctl is-active --quiet caddy; then
    echo "  ✔ Caddy is running"
else
    echo "  ✘ Caddy is NOT running"
    systemctl status caddy --no-pager
fi

# 3. Check autoshutdown timer/service
echo "[*] Checking autoshutdown..."
if systemctl is-enabled --quiet minecraft-autoshutdown.timer; then
    echo "  ✔ autoshutdown timer is enabled"
    systemctl list-timers --all | grep minecraft-autoshutdown || true
else
    echo "  ✘ autoshutdown timer not enabled"
fi

# 4. Check map rebuild timer/service
echo "[*] Checking map rebuild..."
if systemctl is-enabled --quiet minecraft-map-rebuild.timer; then
    echo "  ✔ map-rebuild timer is enabled"
    systemctl list-timers --all | grep minecraft-map-rebuild || true
else
    echo "  ✘ map-rebuild timer not enabled"
fi

# 5. Check directories and symlink
echo "[*] Checking map symlink..."
if [[ -L /var/www/maps ]]; then
    echo "  ✔ /var/www/maps is a symlink → $(readlink -f /var/www/maps)"
else
    echo "  ✘ /var/www/maps is missing or not a symlink"
fi

# 6. Check ports (Minecraft + Caddy)
echo "[*] Checking listening ports..."
ss -tulwn | grep -E '(:25565|:80|:443)' || echo "  ✘ Expected ports not open"

# 7. Optional: query Minecraft via mcrcon (if you have creds)
if [[ -f /etc/minecraft.env ]]; then
    source /etc/minecraft.env
    echo "[*] Querying Minecraft via RCON..."
    if command -v mcrcon >/dev/null; then
        echo "list" | mcrcon -H 127.0.0.1 -P 25575 -p "${RCON_PASSWORD}" || \
          echo "  ✘ mcrcon query failed"
    else
        echo "  (mcrcon not installed)"
    fi
fi

echo "=== Health check complete ==="
EOF

sudo chmod 0755 /usr/local/bin/mc-healthcheck.sh

cat >/etc/systemd/system/minecraft-health.service <<'EOF'
[Unit]
Description=Minecraft stack health check
Wants=network-online.target
After=network-online.target

[Service]
Type=oneshot
ExecStart=/usr/local/bin/mc-healthcheck.sh
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF
