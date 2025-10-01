data "aws_vpc" "default" {
  default = true
}

data "aws_subnets" "default" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.default.id]
  }
}

variable "aws_profile" {
  description = "AWS CLI profile to use"
  type        = string
  default     = "minecraft"
}

provider "aws" {
  region  = "us-east-2"
  profile = var.aws_profile
}

module "mc_stack" {
  source            = "../modules/mc_stack"
  name              = "minecraft-test"
  ami_id            = "ami-033bf9694c0d2ea06"
  instance_type     = "t3.small"
  vpc_id            = data.aws_vpc.default.id
  subnet_id         = tolist(data.aws_subnets.default.ids)[0]
  key_name          = "minecraft-packer"
  root_volume_size  = 8
  ssh_cidr_blocks   = ["104.230.245.46/32"]
  world_version     = "1.21.8"
  availability_zone = "us-east-2b"
}

output "server_public_ip" {
  value = module.mc_stack.public_ip
}

output "server_private_ip" {
  value = module.mc_stack.private_ip
}

output "server_ipv6" {
  value = module.mc_stack.ipv6_addresses
}
