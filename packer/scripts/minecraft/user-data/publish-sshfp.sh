#!/usr/bin/env bash
set -euo pipefail

# Publish SSH host key fingerprints (SSHFP) to Route53
# Requires: ROUTE53_ZONE_ID and ROUTE53_DNS_NAME environment variables
# These can be set in /etc/minecraft.env or passed directly

# Load environment if available
if [ -f /etc/minecraft.env ]; then
  # shellcheck disable=SC1091
  source /etc/minecraft.env
fi

# Check required variables
if [ -z "${ROUTE53_ZONE_ID:-}" ]; then
  echo "ROUTE53_ZONE_ID not set, skipping SSHFP publish"
  exit 0
fi

if [ -z "${ROUTE53_DNS_NAME:-}" ]; then
  echo "ROUTE53_DNS_NAME not set, skipping SSHFP publish"
  exit 0
fi

echo "Publishing SSHFP records for ${ROUTE53_DNS_NAME} to zone ${ROUTE53_ZONE_ID}"

# Generate SSHFP records from host keys
# ssh-keygen -r outputs: hostname IN SSHFP <alg> <fp_type> <fingerprint>
SSHFP_RECORDS=$(ssh-keygen -r "${ROUTE53_DNS_NAME}" -f /etc/ssh/ssh_host_ed25519_key.pub 2>/dev/null || true)

if [ -z "$SSHFP_RECORDS" ]; then
  echo "No SSHFP records generated, host keys may not exist yet"
  exit 1
fi

# Build Route53 resource records JSON array
RESOURCE_RECORDS=""
while IFS= read -r line; do
  # Parse: hostname IN SSHFP <alg> <fp_type> <fingerprint>
  ALG=$(echo "$line" | awk '{print $4}')
  FP_TYPE=$(echo "$line" | awk '{print $5}')
  FINGERPRINT=$(echo "$line" | awk '{print $6}')

  if [ -n "$ALG" ] && [ -n "$FP_TYPE" ] && [ -n "$FINGERPRINT" ]; then
    if [ -n "$RESOURCE_RECORDS" ]; then
      RESOURCE_RECORDS="${RESOURCE_RECORDS},"
    fi
    RESOURCE_RECORDS="${RESOURCE_RECORDS}{\"Value\": \"${ALG} ${FP_TYPE} ${FINGERPRINT}\"}"
  fi
done <<< "$SSHFP_RECORDS"

if [ -z "$RESOURCE_RECORDS" ]; then
  echo "Failed to parse SSHFP records"
  exit 1
fi

# Ensure DNS name ends with a dot for Route53
DNS_NAME_FQDN="${ROUTE53_DNS_NAME}"
if [[ ! "$DNS_NAME_FQDN" =~ \.$ ]]; then
  DNS_NAME_FQDN="${DNS_NAME_FQDN}."
fi

# Build the change batch JSON
CHANGE_BATCH=$(cat <<EOF
{
  "Comment": "SSHFP records for ${ROUTE53_DNS_NAME}",
  "Changes": [
    {
      "Action": "UPSERT",
      "ResourceRecordSet": {
        "Name": "${DNS_NAME_FQDN}",
        "Type": "SSHFP",
        "TTL": 300,
        "ResourceRecords": [${RESOURCE_RECORDS}]
      }
    }
  ]
}
EOF
)

# Submit to Route53
echo "Submitting SSHFP records to Route53..."
aws route53 change-resource-record-sets \
  --hosted-zone-id "${ROUTE53_ZONE_ID}" \
  --change-batch "${CHANGE_BATCH}"

echo "SSHFP records published successfully"
