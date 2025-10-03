#!/usr/bin/env bash
set -euxo pipefail

# mount-ebs.sh
sudo tee /usr/local/bin/mount-ebs.sh >/dev/null <<'EOF'
#!/bin/bash
set -euxo pipefail

MOUNT_POINT="${MOUNT_POINT:-/srv/minecraft-server}"

# Wait until at least 2 disks are present (root + data)
for i in $(seq 1 60); do
  DISK_COUNT=$(lsblk -dn -o TYPE | grep -c disk || true)
  [ "$DISK_COUNT" -ge 2 ] && break
  sleep 2
done

ROOT_DEVICE=$(lsblk -no PKNAME "$(findmnt -no SOURCE /)" || true)
DEVICE=$(lsblk -dn -o NAME,TYPE | awk -v root="$ROOT_DEVICE" '$2=="disk" && $1!=root {print "/dev/"$1; exit}')

[ -n "$DEVICE" ] || { echo "No data volume found"; exit 1; }

if ! blkid "$DEVICE" >/dev/null 2>&1; then
  mkfs.xfs -f "$DEVICE"
fi

mkdir -p "$MOUNT_POINT"
UUID=$(blkid -s UUID -o value "$DEVICE")

grep -q "$UUID" /etc/fstab || echo "UUID=$UUID $MOUNT_POINT xfs defaults,nofail 0 2" | sudo tee -a /etc/fstab
mount -a
EOF
sudo chmod 0755 /usr/local/bin/mount-ebs.sh

# setup-env.sh
sudo tee /usr/local/bin/setup-env.sh >/dev/null <<'EOS'
#!/bin/bash
set -euxo pipefail

MOUNT_POINT="${MOUNT_POINT:-/srv/minecraft-server}"
CONFIG_FILE="$MOUNT_POINT/minecraft.env"

if [ -f "$CONFIG_FILE" ]; then
  echo "Reusing existing $CONFIG_FILE"
  rm -f /etc/minecraft.env
  ln -s "$CONFIG_FILE" /etc/minecraft.env
elif [ -f /etc/minecraft.env ]; then
  echo "Migrating /etc/minecraft.env -> $CONFIG_FILE"
  mv /etc/minecraft.env "$CONFIG_FILE"
  ln -s "$CONFIG_FILE" /etc/minecraft.env
else
  echo "Generating new $CONFIG_FILE"
  cat > "$CONFIG_FILE" <<EOF
RCON_PASS=$(openssl rand -hex 16)
RCON_PORT=25575
EOF
  ln -s "$CONFIG_FILE" /etc/minecraft.env
fi
EOS
sudo chmod 0755 /usr/local/bin/setup-env.sh

# setup-maps.sh
sudo tee /usr/local/bin/setup-maps.sh >/dev/null <<'EOF'
#!/bin/bash
set -euxo pipefail

mkdir -p /srv/minecraft-server/maps
if [ ! -L /var/www/map ]; then
    rm -rf /var/www/map
    ln -s /srv/minecraft-server/maps /var/www/map
fi
EOF
sudo chmod 0755 /usr/local/bin/setup-maps.sh
