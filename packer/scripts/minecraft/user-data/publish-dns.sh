#!/usr/bin/env bash
set -euo pipefail

# Publish DNS records (A, AAAA, SSHFP) to Route53 on instance boot
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

echo "Publishing DNS records for ${ROUTE53_DNS_NAME} to zone ${ROUTE53_ZONE_ID}"

# Ensure DNS name ends with a dot for Route53
DNS_NAME_FQDN="${ROUTE53_DNS_NAME}"
if [[ ! "$DNS_NAME_FQDN" =~ \.$ ]]; then
  DNS_NAME_FQDN="${DNS_NAME_FQDN}."
fi

# Get instance metadata token (IMDSv2)
TOKEN=$(curl -sX PUT "http://169.254.169.254/latest/api/token" \
  -H "X-aws-ec2-metadata-token-ttl-seconds: 60")

# Get public IPv4 address
PUBLIC_IPV4=$(curl -sf -H "X-aws-ec2-metadata-token: $TOKEN" \
  "http://169.254.169.254/latest/meta-data/public-ipv4" 2>/dev/null || echo "")

# Get IPv6 address (if available)
IPV6_ADDR=$(curl -sf -H "X-aws-ec2-metadata-token: $TOKEN" \
  "http://169.254.169.254/latest/meta-data/ipv6" 2>/dev/null || echo "")

# Build changes array
CHANGES=""

# Add A record if we have a public IPv4
if [ -n "$PUBLIC_IPV4" ]; then
  echo "Adding A record: ${ROUTE53_DNS_NAME} -> ${PUBLIC_IPV4}"
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
fi

# Add AAAA record if we have IPv6
if [ -n "$IPV6_ADDR" ]; then
  echo "Adding AAAA record: ${ROUTE53_DNS_NAME} -> ${IPV6_ADDR}"
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
fi

# Generate SSHFP records from ED25519 host key
SSHFP_RECORDS=$(ssh-keygen -r "${ROUTE53_DNS_NAME}" -f /etc/ssh/ssh_host_ed25519_key.pub 2>/dev/null || true)

if [ -n "$SSHFP_RECORDS" ]; then
  # Build SSHFP resource records
  SSHFP_RRS=""
  while IFS= read -r line; do
    # Parse: hostname IN SSHFP <alg> <fp_type> <fingerprint>
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
    echo "Adding SSHFP records for ${ROUTE53_DNS_NAME}"
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

# Submit to Route53 if we have any changes
if [ -z "$CHANGES" ]; then
  echo "No DNS records to publish"
  exit 0
fi

CHANGE_BATCH=$(cat <<EOF
{
  "Comment": "Auto-published DNS records for ${ROUTE53_DNS_NAME}",
  "Changes": [${CHANGES}]
}
EOF
)

echo "Submitting DNS records to Route53..."
aws route53 change-resource-record-sets \
  --hosted-zone-id "${ROUTE53_ZONE_ID}" \
  --change-batch "${CHANGE_BATCH}"

echo "DNS records published successfully"
