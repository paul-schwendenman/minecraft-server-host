packer {}

variable "test_host" {
  description = "Public IP or DNS of the test server"
  type        = string
  default     = "localhost"  # Default value for development
}

variable "test_user" {
  description = "SSH username"
  type        = string
  default     = "ubuntu"
}

variable "test_private_key" {
  description = "Private key path"
  type        = string
  default     = "~/.ssh/id_rsa"
}

source "null" "minecraft" {
  communicator         = "ssh"
  ssh_host             = var.test_host
  ssh_username         = var.test_user
  ssh_private_key_file = var.test_private_key
}

build {
  name    = "minecraft-live-test"
  sources = ["source.null.minecraft"]

  provisioner "shell" {
    inline = ["mkdir -p /tmp/scripts"]
  }

  provisioner "file" {
    source      = "scripts/"
    destination = "/tmp/scripts/"
  }

  # provisioner "shell" { script = "scripts/install_deps.sh" }
  # provisioner "shell" { script = "scripts/create-minecraft-user.sh" }
  # provisioner "shell" { script = "scripts/install_minecraft_service.sh" }
  # provisioner "shell" { script = "scripts/install_user_data_helpers.sh" }

  # provisioner "shell" { script = "scripts/install_autoshutdown.sh" }
  # provisioner "shell" { script = "scripts/install_caddy_unmined.sh" }
  # provisioner "shell" { script = "scripts/install_create_world.sh" }
  # provisioner "shell" { script = "scripts/install_map_rebuild.sh" }
  # provisioner "shell" { script = "scripts/install_map_refresh.sh" }
  # provisioner "shell" { script = "scripts/install_map_backup.sh" }
  # provisioner "shell" { script = "scripts/install_world_backup.sh" }
  # provisioner "shell" { script = "scripts/install_mc_healthcheck.sh" }

  provisioner "shell" {
    inline = [
      "sudo systemctl daemon-reexec",
      "sudo systemctl daemon-reload",
      "sudo apt-get clean",
      "sudo rm -rf /tmp/* /var/tmp/*"
    ]
  }
}
