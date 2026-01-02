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

variable "restic_password" {
  description = "Password for restic backup encryption"
  type        = string
  sensitive   = true
}

provider "aws" {
  region  = "us-east-2"
  profile = var.aws_profile
}

# us-east-1 provider required for ACM certificates used by CloudFront
provider "aws" {
  alias   = "us_east_1"
  region  = "us-east-1"
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

# ACM certificate for CloudFront custom domains (must be in us-east-1)
module "acm_certificate" {
  source = "../modules/acm_certificate"
  providers = {
    aws = aws.us_east_1
  }

  domain  = "*.minecraft.paulandsierra.com"
  zone_id = aws_route53_zone.prod.zone_id
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
  backup_bucket        = module.s3_buckets.backup_bucket_name
  restic_password      = var.restic_password
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

# Manager app (www.*)
module "web_ui_www" {
  source = "../modules/web_ui"
  name   = "minecraft-prod-www"

  api_endpoint              = module.api_lambda.api_endpoint
  webapp_bucket_name        = module.s3_buckets.webapp_bucket_name
  webapp_bucket_domain_name = module.s3_buckets.webapp_bucket_domain_name

  custom_domain       = "www.minecraft.paulandsierra.com"
  acm_certificate_arn = module.acm_certificate.certificate_arn
  zone_id             = aws_route53_zone.prod.zone_id

  include_maps  = false
  geo_whitelist = []
}

# Worlds app (maps.*)
module "web_ui_maps" {
  source = "../modules/web_ui"
  name   = "minecraft-prod-maps"

  api_endpoint              = module.api_lambda.api_endpoint
  webapp_bucket_name        = module.s3_buckets.webapp_maps_bucket_name
  webapp_bucket_domain_name = module.s3_buckets.webapp_maps_bucket_domain_name
  map_bucket_name           = module.s3_buckets.map_bucket_name
  map_bucket_domain_name    = module.s3_buckets.map_bucket_domain_name

  custom_domain       = "maps.minecraft.paulandsierra.com"
  acm_certificate_arn = module.acm_certificate.certificate_arn
  zone_id             = aws_route53_zone.prod.zone_id

  include_maps  = true
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

output "webapp_www_url" {
  value = module.web_ui_www.webapp_url
}

output "webapp_maps_url" {
  value = module.web_ui_maps.webapp_url
}

output "cloudfront_www_distribution_id" {
  value = module.web_ui_www.distribution_id
}

output "cloudfront_maps_distribution_id" {
  value = module.web_ui_maps.distribution_id
}

output "webapp_bucket_name" {
  value = module.s3_buckets.webapp_bucket_name
}

output "webapp_maps_bucket_name" {
  value = module.s3_buckets.webapp_maps_bucket_name
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
