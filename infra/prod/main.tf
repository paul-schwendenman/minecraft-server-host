variable "aws_profile" {
  description = "AWS CLI profile to use"
  type        = string
  default     = "minecraft"
}

variable "ami_id" {
  description = "AMI ID for the Minecraft EC2 instance. If null, uses the latest minecraft-ubuntu-* AMI."
  type        = string
  default     = null
}

provider "aws" {
  region  = "us-east-2"
  profile = var.aws_profile
}

# Automatically find the most recent Minecraft AMI
data "aws_ami" "minecraft" {
  most_recent = true
  owners      = ["self"]

  filter {
    name   = "name"
    values = ["minecraft-ubuntu-*"]
  }
}

# Route53 zone - managed by prod (imported from old setup)
resource "aws_route53_zone" "prod" {
  name = "minecraft.paulandsierra.com"
}

module "networking" {
  source             = "../modules/networking"
  name               = "minecraft-prod"
  vpc_cidr           = "10.1.0.0/16"
  public_subnet_cidr = "10.1.1.0/24"
  availability_zone  = "us-east-2c"
}

module "s3_buckets" {
  source = "../modules/s3_buckets"
  name   = "minecraft-prod"
}

module "ec2_role" {
  source        = "../modules/ec2_role"
  name          = "minecraft-prod"
  map_bucket    = module.s3_buckets.map_bucket_name
  backup_bucket = module.s3_buckets.backup_bucket_name
}

module "mc_stack" {
  source           = "../modules/mc_stack"
  name             = "minecraft-prod"
  ami_id           = var.ami_id != null ? var.ami_id : data.aws_ami.minecraft.id
  instance_type    = "t3.medium"
  vpc_id           = module.networking.vpc_id
  subnet_id        = module.networking.public_subnet_id
  key_name         = "minecraft-packer"
  root_volume_size = 8
  data_volume_size = 8
  ssh_cidr_blocks = [
    "104.230.245.46/32",
  ]
  world_version     = "1.21.4"
  availability_zone = "us-east-2c"

  iam_instance_profile = module.ec2_role.instance_profile_name
  map_bucket           = module.s3_buckets.map_bucket_name
}

module "api_lambda" {
  source          = "../modules/api_lambda"
  name            = "minecraft-prod"
  instance_id     = module.mc_stack.instance_id
  instance_arn    = module.mc_stack.instance_arn
  dns_name        = aws_route53_zone.prod.name
  zone_id         = aws_route53_zone.prod.zone_id
  map_bucket_name = module.s3_buckets.map_bucket_name
  cors_origin     = "*"
}

module "web_ui" {
  source = "../modules/web_ui"
  name   = "minecraft-prod"

  api_endpoint              = module.api_lambda.api_endpoint
  webapp_bucket_name        = module.s3_buckets.webapp_bucket_name
  webapp_bucket_domain_name = module.s3_buckets.webapp_bucket_domain_name
  map_bucket_name           = module.s3_buckets.map_bucket_name
  map_bucket_domain_name    = module.s3_buckets.map_bucket_domain_name

  # No geo restrictions for production
  geo_whitelist = []
}

module "dns_records" {
  source = "../modules/dns_records"

  zone_id        = aws_route53_zone.prod.zone_id
  dns_name       = aws_route53_zone.prod.name
  ipv4_addresses = [module.mc_stack.public_ip]
  ipv6_addresses = module.mc_stack.ipv6_addresses

  create_a_record    = true
  create_aaaa_record = true
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

output "api_endpoint" {
  value = module.api_lambda.api_endpoint
}

output "webapp_url" {
  value = module.web_ui.webapp_url
}

output "map_bucket_name" {
  value = module.s3_buckets.map_bucket_name
}

output "latest_ami_id" {
  description = "The latest available minecraft-ubuntu-* AMI ID"
  value       = data.aws_ami.minecraft.id
}

output "latest_ami_name" {
  description = "The name of the latest available minecraft-ubuntu-* AMI"
  value       = data.aws_ami.minecraft.name
}

output "current_ami_id" {
  description = "The AMI ID currently configured for the EC2 instance"
  value       = var.ami_id != null ? var.ami_id : data.aws_ami.minecraft.id
}
