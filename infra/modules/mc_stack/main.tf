resource "aws_security_group" "minecraft" {
  name        = "${var.name}-sg"
  description = "Allow Minecraft and SSH"
  vpc_id      = var.vpc_id

  ingress {
    description      = "Minecraft"
    from_port        = 25565
    to_port          = 25565
    protocol         = "tcp"
    cidr_blocks      = ["0.0.0.0/0"]
    ipv6_cidr_blocks = ["::/0"]
  }

  ingress {
    description = "SSH"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = var.ssh_cidr_blocks
  }

  egress {
    from_port        = 0
    to_port          = 0
    protocol         = "-1"
    cidr_blocks      = ["0.0.0.0/0"]
    ipv6_cidr_blocks = ["::/0"]
  }
}

resource "aws_instance" "minecraft" {
  ami                         = var.ami_id
  instance_type               = var.instance_type
  subnet_id                   = var.subnet_id
  vpc_security_group_ids      = [aws_security_group.minecraft.id]
  key_name                    = var.key_name
  associate_public_ip_address = true
  availability_zone           = var.availability_zone

  root_block_device {
    volume_size = var.root_volume_size
    volume_type = "gp3"
  }

  user_data = <<-EOT
              #!/bin/bash
              set -euxo pipefail

              MOUNT_POINT="/srv/minecraft-server"

              # Initialization wait
              sleep 30

              # Detect Root to Exclude
              get_root_device() {
                  root_mount=$(findmnt -n -o SOURCE /)
                  echo "${root_mount%p*}"  # Removes partition suffix (p1---pn) --> To get Base Disk
              }

              ROOT_DEVICE=$(get_root_device)

              DEVICE=""

              # Check NVMe device first & Exclude ROOT
              if ls /dev/nvme* 1>/dev/null 2>&1; then
                  DEVICE=$(lsblk -n -p -o NAME,TYPE | grep "disk" | grep -E "/dev/nvme[0-9]" | awk '{print $1}' | grep -v "${ROOT_DEVICE}" | head -1)
              fi

              # No NVMe, check for xvd devices
              if [ -z "${DEVICE}" ]; then
                  DEVICE=$(lsblk -n -p -o NAME,TYPE | grep "disk" | grep -E "/dev/xvd[f-p]" | awk '{print $1}' | head -1)
              fi

              # No NVMe || xvd, check for sd devices
              if [ -z "${DEVICE}" ]; then
                  DEVICE=$(lsblk -n -p -o NAME,TYPE | grep "disk" | grep -E "/dev/sd[f-p]" | awk '{print $1}' | head -1)
              fi

              # None found --> Exit
              if [ -z "${DEVICE}" ]; then
                  echo "No suitable EBS volume found"
                  exit 1
              fi

              # Format the device --> Create Filesystem
              if ! blkid "${DEVICE}" >/dev/null 2>&1; then
                  echo "Creating filesystem on ${DEVICE}"
                  mkfs.ext4 "${DEVICE}"
              fi

              # Create MountPoint Dircetory
              mkdir -p "${MOUNT_POINT}"

              # Retrieve UUID of the device
              UUID=$(blkid -s UUID -o value "${DEVICE}")
              echo "Device UUID: $UUID"

              # input fstab Entry
              if ! grep -q "$UUID" /etc/fstab; then
                  echo "UUID=$UUID ${MOUNT_POINT} ext4 defaults,nofail 0 2" >> /etc/fstab
              fi

              # Check Conflicting mounts (if already occur)
              if mountpoint -q "${MOUNT_POINT}"; then
                  umount "${MOUNT_POINT}"
              fi

              # Mount
              mount "${MOUNT_POINT}" || mount -a


              # Set Permissions
              chown -R ubuntu:ubuntu "${MOUNT_POINT}"
              chmod 755 "${MOUNT_POINT}"

              /usr/local/bin/create-world.sh ${var.world_name} ${var.world_version} ${var.world_seed}

              mkdir -p /srv/minecraft-server/maps
              if [ ! -L /var/www/maps ]; then
                rm -rf /var/www/maps
                ln -s /srv/minecraft-server/maps /var/www/maps
              fi
              EOT

  tags = {
    Name = "${var.name}-server"
  }
}

resource "aws_ebs_volume" "world" {
  availability_zone = var.availability_zone
  size              = var.data_volume_size
  type              = "gp3"

  tags = {
    Name = "${var.name}-world-data"
  }
  lifecycle {
    prevent_destroy = true
  }
}

resource "aws_volume_attachment" "world_attachment" {
  device_name = var.data_volume_device_name
  volume_id   = aws_ebs_volume.world.id
  instance_id = aws_instance.minecraft.id
}
