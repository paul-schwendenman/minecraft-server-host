provider "aws" {
  region  = var.aws_region
  profile = var.aws_profile
}

# Security group to allow SSH + Minecraft + HTTP (for Caddy)
resource "aws_security_group" "minecraft" {
  name        = "minecraft-test-sg"
  description = "Allow SSH, Minecraft, and HTTP"
  vpc_id      = var.vpc_id

  ingress {
    description = "SSH"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    description = "Minecraft"
    from_port   = 25565
    to_port     = 25565
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    description = "HTTP for Caddy map"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_instance" "minecraft" {
  ami                    = var.ami_id
  instance_type          = var.instance_type
  subnet_id              = var.subnet_id
  vpc_security_group_ids = [aws_security_group.minecraft.id]
  key_name               = var.key_name

  root_block_device {
    volume_size = 16
  }

  user_data = <<-EOT
              #!/bin/bash
              /usr/local/bin/create-world.sh ${var.world_name} ${var.world_version} ${var.world_seed}
              EOT

  tags = {
    Name = "minecraft-test"
  }
}
