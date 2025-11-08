packer {
  required_plugins {
    virtualbox = {
      version = ">= 1.0.0"
      source  = "github.com/hashicorp/virtualbox"
    }
  }
}

source "virtualbox-iso" "ubuntu_vm" {
  iso_url          = "https://releases.ubuntu.com/22.04/ubuntu-22.04.5-live-server-amd64.iso"
  iso_checksum     = "auto"
  ssh_username     = "ubuntu"
  ssh_password     = "ubuntu"
  shutdown_command = "echo 'ubuntu' | sudo -S shutdown -P now"
  guest_os_type    = "Ubuntu_64"
}

build {
  name    = "minecraft-vm"
  sources = ["source.virtualbox-iso.ubuntu_vm"]

  provisioner "shell" {
    script = "scripts/install_deps.sh"
  }
}
