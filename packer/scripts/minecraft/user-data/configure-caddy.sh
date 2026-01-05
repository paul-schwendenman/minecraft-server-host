#!/usr/bin/env bash
set -euo pipefail

# Configure Caddy with HTTPS for a given domain
# Uses HTTP-01 ACME challenge (requires port 80 open)
#
# Usage: configure-caddy.sh <domain>
# Example: configure-caddy.sh minecraft.example.com

DOMAIN="${1:-}"
MAP_DIR="/var/www/map"

if [ -z "$DOMAIN" ]; then
  echo "No domain provided, skipping Caddy HTTPS configuration"
  exit 0
fi

echo "Configuring Caddy for HTTPS on ${DOMAIN}"

sudo tee /etc/caddy/Caddyfile >/dev/null <<EOF
${DOMAIN} {
    root * ${MAP_DIR}
    file_server
}
EOF

sudo systemctl reload caddy

echo "Caddy configured for ${DOMAIN}"
