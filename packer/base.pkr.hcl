variable "minecraft_jars" {
  type = list(object({
    version = string
    url     = string
    sha256  = string
  }))
  default     = []
  description = "List of Minecraft JARs to download in base AMI"
}

# version/sha256 come from unmined.auto.pkrvars.hcl, which is checked in and
# updated by .github/workflows/unmined-update.yml.
variable "unmined_cli" {
  type = object({
    version = string
    sha256  = string
  })
  description = "Pinned unmined-cli version/hash. See docs/plans/unmined-cli-docs-plan.md."
}

# A presigned S3 GetObject URL for that exact version/hash, valid for the
# duration of the build. Not committed: generate it with
# `scripts/unmined-artifact.sh presign <version> <sha256>` and pass it with
# -var, since the build instance itself has no AWS credentials.
variable "unmined_cli_url" {
  type        = string
  description = "Presigned URL for the pinned unmined-cli tarball in the artifacts bucket"
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
    environment_vars = [
      "UNMINED_URL=${var.unmined_cli_url}",
      "UNMINED_SHA256=${var.unmined_cli.sha256}",
    ]
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
