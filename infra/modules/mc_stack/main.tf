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
              DEVICE="/dev/disk/by-id/nvme-Amazon_Elastic_Block_Store_${aws_ebs_volume.world.id}"

              # wait for the symlink
              for i in $(seq 1 60); do
                [ -e "$DEVICE" ] && break
                sleep 2
              done
              [ -e "$DEVICE" ] || { echo "ERROR: device $DEVICE not found"; exit 1; }

              # format if needed
              if ! blkid -o value -s TYPE "$DEVICE" >/dev/null 2>&1; then
                mkfs.xfs -f "$DEVICE"
              fi

              mkdir -p "$MOUNT_POINT"

              UUID="$(blkid -s UUID -o value "$DEVICE")"
              grep -q "$UUID" /etc/fstab || echo "UUID=$UUID $MOUNT_POINT xfs defaults,nofail 0 2" >> /etc/fstab

              mount -a

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
