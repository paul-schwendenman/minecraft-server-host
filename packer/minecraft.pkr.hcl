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
    sha256  = string
  }))
  default     = []
  description = "List of Minecraft JARs to download in base AMI"
}

source "amazon-ebs" "minecraft" {
  region        = "us-east-2"
  instance_type = "t3a.medium"
  # source_ami_filter {
  #   filters = {
  #     name                = "ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*"
  #     root-device-type    = "ebs"
  #     virtualization-type = "hvm"
  #   }
  #   owners      = ["099720109477"] # Canonical
  #   most_recent = true
  # }

  source_ami_filter {
    filters = {
      name = "minecraft-base-*"
    }
    owners      = ["self"]
    most_recent = true
  }
  ssh_username = "ubuntu"
  ami_name     = "minecraft-ubuntu-{{timestamp}}"
}

build {
  name    = "minecraft-ami"
  sources = ["source.amazon-ebs.minecraft"]

  # --------------------------------------------------------------------------
  # 1. Scripts + Minecraft-specific tools
  # --------------------------------------------------------------------------
  provisioner "shell" {
    inline = ["mkdir -p /tmp/scripts"]
  }

  provisioner "file" {
    source      = "scripts/"
    destination = "/tmp/scripts/"
  }

  # Install minecraftctl (internal CLI)
  provisioner "shell" { script = "scripts/minecraft/install_minecraftctl.sh" }

  provisioner "shell" {
    inline = [
      "RCON_PASS=$(openssl rand -hex 16)",
      "echo \"RCON_PASSWORD=$RCON_PASS\" | sudo tee /etc/minecraft.env",
      "echo \"RCON_PORT=25575\" | sudo tee -a /etc/minecraft.env",
      "sudo chmod 600 /etc/minecraft.env",
      "sudo chown root:root /etc/minecraft.env"
    ]
  }

  # --------------------------------------------------------------------------
  # 3. Minecraft Service + Core Helpers
  # --------------------------------------------------------------------------
  provisioner "shell" { script = "scripts/minecraft/install_minecraft_service.sh" }
  provisioner "shell" { script = "scripts/minecraft/install_user_data_helpers.sh" }

  # --------------------------------------------------------------------------
  # 4. Install modular script groups
  # --------------------------------------------------------------------------
  provisioner "shell" { script = "scripts/minecraft/install_autoshutdown.sh" }
  provisioner "shell" { script = "scripts/minecraft/install_create_world.sh" }
  provisioner "shell" { script = "scripts/minecraft/install_map_build.sh" }
  provisioner "shell" { script = "scripts/minecraft/install_map_backup.sh" }
  provisioner "shell" { script = "scripts/minecraft/install_world_backup.sh" }
  provisioner "shell" { script = "scripts/minecraft/install_mc_healthcheck.sh" }

  # --------------------------------------------------------------------------
  # 5. Install Minecraft JARs
  # --------------------------------------------------------------------------
  provisioner "shell" {
    environment_vars = [
      "MINECRAFT_JARS_JSON=${jsonencode(var.minecraft_jars)}"
    ]
    script = "scripts/shared/install_minecraft_jars.sh"
  }

  # --------------------------------------------------------------------------
  # 6. Final system prep
  # --------------------------------------------------------------------------
  provisioner "shell" {
    inline = [
      "sudo systemctl daemon-reexec",
      "sudo systemctl daemon-reload",
      "sudo apt-get clean",
      "sudo rm -rf /tmp/* /var/tmp/*"
    ]
  }
}
