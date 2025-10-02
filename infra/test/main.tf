variable "aws_profile" {
  description = "AWS CLI profile to use"
  type        = string
  default     = "minecraft"
}

provider "aws" {
  region  = "us-east-2"
  profile = var.aws_profile
}

module "networking" {
  source             = "../modules/networking"
  name               = "minecraft-test"
  vpc_cidr           = "10.0.0.0/16"
  public_subnet_cidr = "10.0.1.0/24"
}

module "mc_stack" {
  source            = "../modules/mc_stack"
  name              = "minecraft-test"
  ami_id            = "ami-09b1684e1740cf826"
  instance_type     = "t3.small"
  vpc_id            = module.networking.vpc_id
  subnet_id         = module.networking.public_subnet_id
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
