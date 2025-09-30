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

  root_block_device {
    volume_size = var.root_volume_size
    volume_type = "gp3"
  }

  user_data = <<-EOT
              #!/bin/bash
              set -eux

              DEVICE="${var.data_volume_device_name}"
              MOUNT_POINT="/srv/minecraft-server/"

              # wait for device
              while [ ! -e $DEVICE ]; do sleep 1; done

              # format if needed
              if ! file -s $DEVICE | grep -q "filesystem"; then
                mkfs.xfs -f $DEVICE
              fi

              mkdir -p $MOUNT_POINT
              mount $DEVICE $MOUNT_POINT
              echo "$DEVICE $MOUNT_POINT xfs defaults,nofail 0 2" >> /etc/fstab

              # hand off to world creation script (will create world inside $MOUNT_POINT)
              /usr/local/bin/create-world.sh ${var.world_name} ${var.world_version} ${var.world_seed}
              EOT

  tags = {
    Name = "${var.name}-server"
  }
}

resource "aws_ebs_volume" "world" {
  availability_zone = aws_instance.minecraft.availability_zone
  size              = var.data_volume_size
  type              = "gp3"

  tags = {
    Name = "${var.name}-world-data"
  }
}

resource "aws_volume_attachment" "world_attachment" {
  device_name = var.data_volume_device_name
  volume_id   = aws_ebs_volume.world.id
  instance_id = aws_instance.minecraft.id
}
