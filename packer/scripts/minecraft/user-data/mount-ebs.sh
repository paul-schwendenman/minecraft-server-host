#!/usr/bin/env bash
set -euxo pipefail

MOUNT_POINT="${MOUNT_POINT:-/srv/minecraft-server}"

# Wait until at least 2 disks are present (root + data)
for _ in $(seq 1 60); do
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

grep -q "$UUID" /etc/fstab || echo "UUID=$UUID $MOUNT_POINT auto defaults,nofail 0 2" | sudo tee -a /etc/fstab
mount -a
