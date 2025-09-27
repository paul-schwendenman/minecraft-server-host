packer {
  required_plugins {
    amazon = {
      version = ">= 1.3.0"
      source  = "github.com/hashicorp/amazon"
    }
  }
}

variable "minecraft_jars" {
  type = list(object({
    version = string
    url     = string
    sha1    = string
  }))
  default = [
    {
      version = "1.18.1"
      url     = "https://launcher.mojang.com/v1/objects/125e5adf40c659fd3bce3e66e67a16bb49ecc1b9/server.jar"
      sha1    = "ebcd120ad81480b968a548df6ffb83b88075e95195c8ff63d461c9df4df5dbdf"
    }
  ]
}

source "amazon-ebs" "minecraft" {
  region           = "us-east-2"
  instance_type    = "t3a.medium"
  source_ami_filter {
    filters = {
      name                = "ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*"
      root-device-type    = "ebs"
      virtualization-type = "hvm"
    }
    owners      = ["099720109477"] # Canonical
    most_recent = true
  }
  ssh_username     = "ubuntu"
  ami_name         = "minecraft-ubuntu-{{timestamp}}"
}

build {
  name    = "minecraft-ami"
  sources = ["source.amazon-ebs.minecraft"]

  provisioner "shell" { script = "scripts/install_deps.sh" }
  provisioner "shell" { script = "scripts/install_systemd.sh" }

  provisioner "shell" {
    inline = ["sudo mkdir -p /opt/minecraft/jars"]
  }

  provisioner "shell" {
    inline = [
      %{ for jar in var.minecraft_jars ~}
      "echo 'Installing Minecraft ${jar.version}'",
      "curl -fsSL ${jar.url} -o /opt/minecraft/jars/minecraft_server_${jar.version}.jar",
      "echo '${jar.sha1}  /opt/minecraft/jars/minecraft_server_${jar.version}.jar' | sha1sum --check -",
      %{ endfor ~}
      "sudo chown -R root:root /opt/minecraft/jars",
      "sudo chmod 755 /opt/minecraft/jars"
    ]
  }

  provisioner "shell" { script = "scripts/install_autoshutdown.sh" }
  provisioner "shell" { script = "scripts/install_caddy_unmined.sh" }
}
