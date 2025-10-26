packer {
  required_plugins {
    docker = {
      source  = "github.com/hashicorp/docker"
      version = "~> 1"
    }
  }
}

source "docker" "ubuntu_local" {
  image  = "ubuntu:22.04"
  commit = true
}

build {
  name    = "minecraft-local"
  sources = ["source.docker.ubuntu_local"]

  provisioner "shell" {
    inline = [
      "apt-get update -qq",
      "apt-get install -y sudo curl wget unzip gnupg software-properties-common"
    ]
  }

  provisioner "file" {
    source      = "scripts/"
    destination = "/tmp/scripts/"
  }

  provisioner "shell" {
    script = "scripts/install_deps.sh"
  }

  post-processor "docker-tag" {
    repository = "minecraft-local"
    tags       = ["latest"]
  }
}
