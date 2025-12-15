variable "minecraft_jars" {
  type = list(object({
    version = string
    url     = string
    sha256  = string
  }))
  default     = []
  description = "List of Minecraft JARs to download in base AMI"
}

source "amazon-ebs" "ubuntu_base" {
  region        = "us-east-2"
  instance_type = "t3.micro"
  source_ami_filter {
    filters = {
      name                = "ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*"
      root-device-type    = "ebs"
      virtualization-type = "hvm"
    }
    owners      = ["099720109477"]
    most_recent = true
  }
  ssh_username = "ubuntu"
  ami_name     = "minecraft-base-{{timestamp}}"
}

build {
  name    = "minecraft-base"
  sources = ["source.amazon-ebs.ubuntu_base"]

  provisioner "shell" {
    inline = ["mkdir -p /tmp/scripts"]
  }

  provisioner "file" {
    source      = "scripts/"
    destination = "/tmp/scripts/"
  }

  provisioner "shell" {
    script = "scripts/base/install_base_deps.sh"
  }

  provisioner "shell" {
    script = "scripts/base/create-minecraft-user.sh"
  }

  # Install Minecraft JARs if provided
  provisioner "shell" {
    environment_vars = [
      "MINECRAFT_JARS_JSON=${jsonencode(var.minecraft_jars)}"
    ]
    script = "scripts/shared/install_minecraft_jars.sh"
  }

  provisioner "shell" {
    inline = [
      "sudo systemctl daemon-reexec",
      "sudo systemctl daemon-reload",
      "sudo apt-get clean",
      "sudo rm -rf /tmp/* /var/tmp/*"
    ]
  }
}
