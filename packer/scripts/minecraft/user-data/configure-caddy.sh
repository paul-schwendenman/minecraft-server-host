#!/usr/bin/env bash
set -euo pipefail

# Configure Caddy with HTTPS for a given domain
# Uses HTTP-01 ACME challenge (requires port 80 open)
# Stores certificates on EBS volume to persist across reprovisioning
#
# Usage: configure-caddy.sh <domain>
# Example: configure-caddy.sh minecraft.example.com

DOMAIN="${1:-}"
MAP_DIR="/var/www/map"
CADDY_DATA_DIR="/srv/minecraft-server/caddy"

if [ -z "$DOMAIN" ]; then
  echo "No domain provided, skipping Caddy HTTPS configuration"
  exit 0
fi

echo "Configuring Caddy for HTTPS on ${DOMAIN}"

# Create persistent storage directory on EBS volume
sudo mkdir -p "${CADDY_DATA_DIR}"
sudo chown caddy:caddy "${CADDY_DATA_DIR}"

sudo tee /etc/caddy/Caddyfile >/dev/null <<EOF
{
    storage file_system ${CADDY_DATA_DIR}
}

${DOMAIN} {
    root * ${MAP_DIR}
    file_server
}
EOF

sudo systemctl reload caddy

echo "Caddy configured for ${DOMAIN} (certs stored in ${CADDY_DATA_DIR})"
