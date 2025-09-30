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

  tags = {
    Name = "${var.name}-server"
  }
}

# Persistent EBS volume for Minecraft world
resource "aws_ebs_volume" "world" {
  availability_zone = aws_instance.minecraft.availability_zone
  size              = var.data_volume_size
  type              = "gp3"

  tags = {
    Name = "${var.name}-world-data"
  }
}

# Attach the volume to the instance
resource "aws_volume_attachment" "world_attachment" {
  device_name = var.data_volume_device_name
  volume_id   = aws_ebs_volume.world.id
  instance_id = aws_instance.minecraft.id
}
