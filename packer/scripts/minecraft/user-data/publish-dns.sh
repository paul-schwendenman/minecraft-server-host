#!/usr/bin/env bash
set -euo pipefail

# Dynamic DNS updater for Route53 (A, AAAA, SSHFP records)
# Only updates records when values have changed.
# Can be run on boot and/or periodically via cron/systemd timer.
#
# Requires: ROUTE53_ZONE_ID and ROUTE53_DNS_NAME environment variables
# These can be set in /etc/minecraft.env or passed directly

# Load environment if available
if [ -f /etc/minecraft.env ]; then
  # shellcheck disable=SC1091
  source /etc/minecraft.env
fi

# Check required variables
if [ -z "${ROUTE53_ZONE_ID:-}" ]; then
  echo "ROUTE53_ZONE_ID not set, skipping DNS publish"
  exit 0
fi

if [ -z "${ROUTE53_DNS_NAME:-}" ]; then
  echo "ROUTE53_DNS_NAME not set, skipping DNS publish"
  exit 0
fi

# Ensure DNS name ends with a dot for Route53
DNS_NAME_FQDN="${ROUTE53_DNS_NAME}"
if [[ ! "$DNS_NAME_FQDN" =~ \.$ ]]; then
  DNS_NAME_FQDN="${DNS_NAME_FQDN}."
fi

# Get current DNS values (using a public resolver to avoid caching issues)
CURRENT_A=$(dig +short A "${ROUTE53_DNS_NAME}" @8.8.8.8 2>/dev/null | head -1 || echo "")
CURRENT_AAAA=$(dig +short AAAA "${ROUTE53_DNS_NAME}" @8.8.8.8 2>/dev/null | head -1 || echo "")

# Get instance metadata token (IMDSv2)
TOKEN=$(curl -sX PUT "http://169.254.169.254/latest/api/token" \
  -H "X-aws-ec2-metadata-token-ttl-seconds: 60")

# Get public IPv4 address
PUBLIC_IPV4=$(curl -sf -H "X-aws-ec2-metadata-token: $TOKEN" \
  "http://169.254.169.254/latest/meta-data/public-ipv4" 2>/dev/null || echo "")

# Get IPv6 address (if available)
IPV6_ADDR=$(curl -sf -H "X-aws-ec2-metadata-token: $TOKEN" \
  "http://169.254.169.254/latest/meta-data/ipv6" 2>/dev/null || echo "")

# Build changes array - only include records that need updating
CHANGES=""

# Check A record
if [ -n "$PUBLIC_IPV4" ]; then
  if [ "$PUBLIC_IPV4" != "$CURRENT_A" ]; then
    echo "A record changed: ${CURRENT_A:-<none>} -> ${PUBLIC_IPV4}"
    CHANGES=$(cat <<EOF
{
  "Action": "UPSERT",
  "ResourceRecordSet": {
    "Name": "${DNS_NAME_FQDN}",
    "Type": "A",
    "TTL": 60,
    "ResourceRecords": [{"Value": "${PUBLIC_IPV4}"}]
  }
}
EOF
)
  else
    echo "A record unchanged: ${PUBLIC_IPV4}"
  fi
fi

# Check AAAA record
if [ -n "$IPV6_ADDR" ]; then
  if [ "$IPV6_ADDR" != "$CURRENT_AAAA" ]; then
    echo "AAAA record changed: ${CURRENT_AAAA:-<none>} -> ${IPV6_ADDR}"
    AAAA_CHANGE=$(cat <<EOF
{
  "Action": "UPSERT",
  "ResourceRecordSet": {
    "Name": "${DNS_NAME_FQDN}",
    "Type": "AAAA",
    "TTL": 60,
    "ResourceRecords": [{"Value": "${IPV6_ADDR}"}]
  }
}
EOF
)
    if [ -n "$CHANGES" ]; then
      CHANGES="${CHANGES},${AAAA_CHANGE}"
    else
      CHANGES="$AAAA_CHANGE"
    fi
  else
    echo "AAAA record unchanged: ${IPV6_ADDR}"
  fi
fi

# SSHFP records - only update on first run or if --force-sshfp is passed
# Check if SSHFP exists by querying DNS
CURRENT_SSHFP=$(dig +short SSHFP "${ROUTE53_DNS_NAME}" @8.8.8.8 2>/dev/null | head -1 || echo "")

if [ -z "$CURRENT_SSHFP" ] || [ "${1:-}" = "--force-sshfp" ]; then
  SSHFP_RECORDS=$(ssh-keygen -r "${ROUTE53_DNS_NAME}" -f /etc/ssh/ssh_host_ed25519_key.pub 2>/dev/null || true)

  if [ -n "$SSHFP_RECORDS" ]; then
    SSHFP_RRS=""
    while IFS= read -r line; do
      ALG=$(echo "$line" | awk '{print $4}')
      FP_TYPE=$(echo "$line" | awk '{print $5}')
      FINGERPRINT=$(echo "$line" | awk '{print $6}')

      if [ -n "$ALG" ] && [ -n "$FP_TYPE" ] && [ -n "$FINGERPRINT" ]; then
        if [ -n "$SSHFP_RRS" ]; then
          SSHFP_RRS="${SSHFP_RRS},"
        fi
        SSHFP_RRS="${SSHFP_RRS}{\"Value\": \"${ALG} ${FP_TYPE} ${FINGERPRINT}\"}"
      fi
    done <<< "$SSHFP_RECORDS"

    if [ -n "$SSHFP_RRS" ]; then
      echo "Updating SSHFP records"
      SSHFP_CHANGE=$(cat <<EOF
{
  "Action": "UPSERT",
  "ResourceRecordSet": {
    "Name": "${DNS_NAME_FQDN}",
    "Type": "SSHFP",
    "TTL": 300,
    "ResourceRecords": [${SSHFP_RRS}]
  }
}
EOF
)
      if [ -n "$CHANGES" ]; then
        CHANGES="${CHANGES},${SSHFP_CHANGE}"
      else
        CHANGES="$SSHFP_CHANGE"
      fi
    fi
  fi
else
  echo "SSHFP record exists, skipping (use --force-sshfp to update)"
fi

# Submit to Route53 only if we have changes
if [ -z "$CHANGES" ]; then
  echo "No DNS changes needed"
  exit 0
fi

CHANGE_BATCH=$(cat <<EOF
{
  "Comment": "Dynamic DNS update for ${ROUTE53_DNS_NAME}",
  "Changes": [${CHANGES}]
}
EOF
)

echo "Submitting DNS changes to Route53..."
aws route53 change-resource-record-sets \
  --hosted-zone-id "${ROUTE53_ZONE_ID}" \
  --change-batch "${CHANGE_BATCH}"

echo "DNS records updated successfully"
