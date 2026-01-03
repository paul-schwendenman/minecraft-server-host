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
  iam_instance_profile        = var.iam_instance_profile
  key_name                    = var.key_name
  associate_public_ip_address = true
  availability_zone           = var.availability_zone
  ipv6_address_count          = 1

  root_block_device {
    volume_size = var.root_volume_size
    volume_type = "gp3"
  }

  lifecycle {
    ignore_changes = [
      associate_public_ip_address,
    ]
  }

  user_data = <<-EOT
              #!/bin/bash
              /usr/local/bin/mount-ebs.sh
              /usr/local/bin/setup-env.sh
              /usr/local/bin/setup-maps.sh
              /usr/local/bin/create-world.sh ${var.world_name} ${var.world_version} ${var.world_seed}
              if ! grep -q '^MC_MAP_BUCKET=' /etc/minecraft.env; then
                echo "MC_MAP_BUCKET=${var.map_bucket}" | sudo tee -a /etc/minecraft.env
              else
                echo "MC_MAP_BUCKET already set, skipping append"
              fi
              if ! grep -q '^MC_WORLD_BUCKET=' /etc/minecraft.env; then
                echo "MC_WORLD_BUCKET=${var.backup_bucket}" | sudo tee -a /etc/minecraft.env
              else
                echo "MC_WORLD_BUCKET already set, skipping append"
              fi
              if ! grep -q '^RESTIC_PASSWORD=' /etc/minecraft.env; then
                echo "RESTIC_PASSWORD=${var.restic_password}" | sudo tee -a /etc/minecraft.env
              else
                echo "RESTIC_PASSWORD already set, skipping append"
              fi
              %{if var.route53_zone_id != "" && var.route53_dns_name != ""}
              export ROUTE53_ZONE_ID="${var.route53_zone_id}"
              export ROUTE53_DNS_NAME="${var.route53_dns_name}"
              /usr/local/bin/publish-dns.sh
              %{endif}
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
