resource "aws_security_group" "minecraft" {
  name        = "${var.name}-sg"
  description = "Allow Minecraft, SSH, and Web"
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

  ingress {
    description      = "HTTP"
    from_port        = 80
    to_port          = 80
    protocol         = "tcp"
    cidr_blocks      = ["0.0.0.0/0"]
    ipv6_cidr_blocks = ["::/0"]
  }

  ingress {
    description      = "HTTPS"
    from_port        = 443
    to_port          = 443
    protocol         = "tcp"
    cidr_blocks      = ["0.0.0.0/0"]
    ipv6_cidr_blocks = ["::/0"]
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
  ipv6_address_count          = 1

  root_block_device {
    volume_size = var.root_volume_size
    volume_type = "gp3"
  }

  user_data = <<-EOT
              #!/bin/bash
              # volumes.sh for Minecraft
              set -euxo pipefail

              MOUNT_POINT="/srv/minecraft-server"
              CONFIG_FILE="$MOUNT_POINT/minecraft.env"

              # Wait up to 2 min for block devices to appear
              for i in $(seq 1 60); do
                DISK_COUNT=$(lsblk -dn -o TYPE | grep -c disk || true)
                if [ "$DISK_COUNT" -ge 2 ]; then
                  break
                fi
                sleep 2
              done

              # Detect root device to exclude
              ROOT_DEVICE=$(lsblk -no PKNAME "$(findmnt -no SOURCE /)" || true)

              DEVICE=""

              # Find first non-root disk (NVMe first)
              DEVICE=$(lsblk -dn -o NAME,TYPE | awk -v root="$ROOT_DEVICE" '$2=="disk" && $1!=root {print "/dev/"$1; exit}')

              if [ -z "$DEVICE" ]; then
                echo "No extra EBS device found"
                exit 1
              fi

              # Format if needed
              if ! blkid "$DEVICE" >/dev/null 2>&1; then
                mkfs.xfs -f "$DEVICE"
              fi

              mkdir -p "$MOUNT_POINT"

              UUID=$(blkid -s UUID -o value "$DEVICE")

              if ! grep -q "$UUID" /etc/fstab; then
                echo "UUID=$UUID $MOUNT_POINT xfs defaults,nofail 0 2" >> /etc/fstab
              fi

              mount -a

              chown -R minecraft:minecraft "$MOUNT_POINT"
              chmod 755 "$MOUNT_POINT"

              /usr/local/bin/create-world.sh ${var.world_name} ${var.world_version} ${var.world_seed}


              # After mounting the EBS volume...

              if [ -f "$CONFIG_FILE" ]; then
                  echo "Reusing existing $CONFIG_FILE"
                  rm -f /etc/minecraft.env
                  ln -s "$CONFIG_FILE" /etc/minecraft.env
              else
                  if [ -f /etc/minecraft.env ]; then
                      echo "Migrating existing /etc/minecraft.env to $CONFIG_FILE"
                      mv /etc/minecraft.env "$CONFIG_FILE"
                  else
                      echo "Generating new $CONFIG_FILE"
                      cat > "$CONFIG_FILE" <<EOF
RCON_PASS=$(openssl rand -hex 16)
RCON_PORT=25575
EOF
                  fi
                  ln -s "$CONFIG_FILE" /etc/minecraft.env
              fi

              mkdir -p /srv/minecraft-server/maps
              if [ ! -L /var/www/map ]; then
                rm -rf /var/www/map
                ln -s /srv/minecraft-server/maps /var/www/map
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
