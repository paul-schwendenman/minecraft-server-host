resource "aws_security_group" "minecraft" {
  name        = var.security_group
  description = "Allow inbound traffic"

  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks     = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 25565
    to_port     = 25565
    protocol    = "tcp"
    cidr_blocks     = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 25565
    to_port     = 25565
    protocol    = "udp"
    cidr_blocks     = ["0.0.0.0/0"]
  }

   egress {
    from_port       = 0
    to_port         = 0
    protocol        = "-1"
    cidr_blocks     = ["0.0.0.0/0"]
  }
}

resource "tls_private_key" "example" {
  algorithm = "RSA"
  rsa_bits  = 4096
}

resource "aws_key_pair" "generated_key" {
  key_name   = var.key_name
  public_key = tls_private_key.example.public_key_openssh
}

resource "local_file" "private_key_pem" {
  depends_on = [tls_private_key.example]

  content    = tls_private_key.example.private_key_pem
  filename   = local.private_key_filename
}

resource "null_resource" "chmod" {
  depends_on = [local_file.private_key_pem]

  provisioner "local-exec" {
    command = "chmod 600 ${local.private_key_filename}"
  }
}

data "aws_ami" "ubuntu" {
  most_recent = true

  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd/ubuntu-bionic-18.04-amd64-server-*"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }

  owners = ["099720109477"] # Canonical
}

resource "aws_instance" "minecraft_server" {
  ami           = data.aws_ami.ubuntu.id
  instance_type = var.instance_type
  key_name = var.key_name
  vpc_security_group_ids = ["${aws_security_group.minecraft.id}"]

  connection {
    type        = "ssh"
    user        = "ubuntu"
    host = aws_instance.minecraft_server.public_ip
    private_key = tls_private_key.example.private_key_pem
  }

  provisioner "remote-exec" {
    script = "./bootstrap.sh"
  }
}

output "instance_id" {
  value = "${aws_instance.minecraft_server.id}"
}

output "ip" {
  value = "${aws_instance.minecraft_server.public_ip}"
}
