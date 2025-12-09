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
  default = [
    {
      version = "1.14.3"
      url     = "https://launcher.mojang.com/v1/objects/d0d0fe2b1dc6ab4c65554cb734270872b72dadd6/server.jar"
      sha256  = "942256f0bfec40f2331b1b0c55d7a683b86ee40e51fa500a2aa76cf1f1041b38"
    },
    {
      version = "1.15.2"
      url     = "https://launcher.mojang.com/v1/objects/bb2b6b1aefcd70dfd1892149ac3a215f6c636b07/server.jar"
      sha256  = "80cf86dc2004ec6a2dc0183d1c75a9af3ba0669f7c332e4247afb1d76fb67e8a"
    },
    {
      version = "1.16.1"
      url     = "https://launcher.mojang.com/v1/objects/a412fd69db1f81db3f511c1463fd304675244077/server.jar"
      sha256  = "2782d547724bc3ffc0ef6e97b2790e75c1df89241f9d4645b58c706f5e6c935b"
    },
    {
      version = "1.16.3"
      url     = "https://launcher.mojang.com/v1/objects/f02f4473dbf152c23d7d484952121db0b36698cb/server.jar"
      sha256  = "32e450e74c081aec06dcfbadfa5ba9aa1c7f370bd869e658caec0c3004f7ad5b"
    },
    {
      version = "1.16.4"
      url     = "https://launcher.mojang.com/v1/objects/35139deedbd5182953cf1caa23835da59ca3d7cd/server.jar"
      sha256  = "444d30d903a1ef489b6737bb9d021494faf23434ca8568fd72ce2e3d40b32506"
    },
    {
      version = "1.18.1"
      url     = "https://launcher.mojang.com/v1/objects/125e5adf40c659fd3bce3e66e67a16bb49ecc1b9/server.jar"
      sha256  = "ebcd120ad81480b968a548df6ffb83b88075e95195c8ff63d461c9df4df5dbdf"
    },
    {
      version = "1.19"
      url     = "https://piston-data.mojang.com/v1/objects/e00c4052dac1d59a1188b2aa9d5a87113aaf1122/server.jar"
      sha256  = "deefd056f0cf89c3d7fd48d03f56a8a73943586e8c061fdabd0fd92d32ced2b2"
    },
    {
      version = "1.19.2"
      url     = "https://piston-data.mojang.com/v1/objects/f69c284232d7c7580bd89a5a4931c3581eae1378/server.jar"
      sha256  = "b26727069ef5f61c704add9a378ac90e3d271fd7876c0bd3dcfbe9fd0bec4d96"
    },
    {
      version = "1.21.8"
      url     = "https://piston-data.mojang.com/v1/objects/6bce4ef400e4efaa63a13d5e6f6b500be969ef81/server.jar"
      sha256  = "2349d9a8f0d4be2c40e7692890ef46a4b07015e7955b075460d02793be7fbbe7"
    },
    {
      version = "1.21.9"
      url     = "https://piston-data.mojang.com/v1/objects/11e54c2081420a4d49db3007e66c80a22579ff2a/server.jar"
      sha256  = "ebc5161abcd57523c14a04ce108f0b671da0d0d74a28926b19a7d139ebabaf31"
    },
    {
      version = "1.21.10"
      url     = "https://piston-data.mojang.com/v1/objects/95495a7f485eedd84ce928cef5e223b757d2f764/server.jar"
      sha256  = "5bb64dc47379903e8f288bd6a4b276e889075c5c0f4c0b714e958d835c1874e7"
    }
  ]
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
  # 1. Base Dependencies + Environment
  # --------------------------------------------------------------------------
  provisioner "shell" {
    inline = ["mkdir -p /tmp/scripts"]
  }

  provisioner "file" {
    source      = "scripts/"
    destination = "/tmp/scripts/"
  }

  provisioner "shell" { script = "scripts/install_deps.sh" }
  provisioner "shell" { script = "scripts/create-minecraft-user.sh" }
  provisioner "shell" { script = "scripts/install_caddy_unmined.sh" }

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
  provisioner "shell" { script = "scripts/install_minecraft_service.sh" }
  provisioner "shell" { script = "scripts/install_user_data_helpers.sh" }

  # --------------------------------------------------------------------------
  # 4. Install modular script groups
  # --------------------------------------------------------------------------
  provisioner "shell" { script = "scripts/install_autoshutdown.sh" }
  provisioner "shell" { script = "scripts/install_create_world.sh" }
  provisioner "shell" { script = "scripts/install_map_rebuild.sh" }
  provisioner "shell" { script = "scripts/install_map_refresh.sh" }
  provisioner "shell" { script = "scripts/install_map_backup.sh" }
  provisioner "shell" { script = "scripts/install_world_backup.sh" }
  provisioner "shell" { script = "scripts/install_mc_healthcheck.sh" }

  # --------------------------------------------------------------------------
  # 5. Install Minecraft JARs
  # --------------------------------------------------------------------------
  provisioner "shell" {
    inline = concat(
      [
        "sudo mkdir -p /opt/minecraft/jars",
        "sudo chown ubuntu:ubuntu /opt/minecraft/jars"
      ],
      [
        for jar in var.minecraft_jars : <<EOC
echo 'Installing Minecraft ${jar.version}'
curl -fsSL ${jar.url} -o /opt/minecraft/jars/minecraft_server_${jar.version}.jar
EOC
      ],
      [
        # Create checksums.txt file in sha256sum format
        "cat > /opt/minecraft/jars/checksums.txt << 'CHECKSUMS_EOF'",
      ],
      [
        for jar in var.minecraft_jars : "${jar.sha256}  minecraft_server_${jar.version}.jar"
      ],
      [
        "CHECKSUMS_EOF",
        # Verify all JARs using checksums.txt
        "cd /opt/minecraft/jars && sha256sum -c checksums.txt",
        "sudo chown -R root:root /opt/minecraft/jars",
        "sudo chmod 755 /opt/minecraft/jars",
        "sudo chmod 644 /opt/minecraft/jars/checksums.txt"
      ]
    )
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
