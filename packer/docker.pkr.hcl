packer {
  required_plugins {
    docker = {
      source  = "github.com/hashicorp/docker"
      version = "~> 1"
    }
  }
}

# version/sha256 come from unmined.auto.pkrvars.hcl, same as base.pkr.hcl.
variable "unmined_cli" {
  type = object({
    version = string
    sha256  = string
  })
  description = "Pinned unmined-cli version/hash. See docs/plans/unmined-cli-docs-plan.md."
}

# Presigned URL, same as base.pkr.hcl: generate with
# `scripts/unmined-artifact.sh presign <version> <sha256>` and pass with
# -var. Needed here too since install_base_deps.sh is shared with base.pkr.hcl.
variable "unmined_cli_url" {
  type        = string
  description = "Presigned URL for the pinned unmined-cli tarball in the artifacts bucket"
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
    environment_vars = [
      "UNMINED_URL=${var.unmined_cli_url}",
      "UNMINED_SHA256=${var.unmined_cli.sha256}",
    ]
    script = "scripts/base/install_base_deps.sh"
  }

  provisioner "shell" {
    script = "scripts/minecraft/install_minecraftctl.sh"
  }

  post-processor "docker-tag" {
    repository = "minecraft-local"
    tags       = ["latest"]
  }
}
