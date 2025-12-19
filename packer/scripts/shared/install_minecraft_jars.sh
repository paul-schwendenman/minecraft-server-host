#!/usr/bin/env bash
set -euxo pipefail

JARS_DIR="/opt/minecraft/jars"
JARS_JSON="${MINECRAFT_JARS_JSON:-}"

if [ -z "$JARS_JSON" ] || [ "$JARS_JSON" = "[]" ]; then
  echo "[!] MINECRAFT_JARS_JSON environment variable not set or empty, skipping JAR installation"
  exit 0
fi

# Ensure jars directory exists and is writable
sudo mkdir -p "${JARS_DIR}"
sudo chown -R ubuntu:ubuntu "${JARS_DIR}"

# Count total JARs to process
TOTAL_JARS=$(echo "$JARS_JSON" | jq '. | length')
echo "[*] Processing ${TOTAL_JARS} Minecraft JAR(s)"

# Download JARs (skip if already exists)
echo "$JARS_JSON" | jq -r '.[] | "\(.version)|\(.url)"' | while IFS='|' read -r version url; do
  JAR_FILE="${JARS_DIR}/minecraft_server_${version}.jar"

  if [ -f "$JAR_FILE" ]; then
    echo "[*] Skipping ${version} - already exists"
  else
    echo "[*] Downloading Minecraft ${version}"
    curl -fsSL "$url" -o "$JAR_FILE"
  fi
done

# Create/update checksums.txt file with all JAR checksums
echo "[*] Creating/updating checksums.txt"
CHECKSUMS_FILE="${JARS_DIR}/checksums.txt"

# If checksums.txt exists, we need to merge with existing entries
# For simplicity, we'll recreate it with all JARs from the JSON
# This ensures all JARs (both existing and new) are verified
echo "$JARS_JSON" | jq -r '.[] | "\(.sha256)  minecraft_server_\(.version).jar"' | sudo tee "$CHECKSUMS_FILE" > /dev/null

# Verify all JARs using checksums.txt with --ignore-missing
# This will verify all JARs that exist, and ignore any that don't exist yet
echo "[*] Verifying JAR checksums"
cd "${JARS_DIR}"
sha256sum -c --ignore-missing checksums.txt || {
  echo "[!] Some JAR checksums failed verification"
  exit 1
}

# Set proper ownership and permissions
sudo chown -R root:root "${JARS_DIR}"
sudo chmod 755 "${JARS_DIR}"
sudo chmod 644 "${JARS_DIR}"/*.jar 2>/dev/null || true
sudo chmod 644 "${JARS_DIR}/checksums.txt"

echo "[*] Minecraft JAR installation complete"

