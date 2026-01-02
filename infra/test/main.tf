variable "aws_profile" {
  description = "AWS CLI profile to use"
  type        = string
  default     = "minecraft"
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

data "aws_route53_zone" "prod" {
  name         = "minecraft.paulandsierra.com."
  private_zone = false
}

# ACM certificate for test environment custom domains (must be in us-east-1)
module "acm_certificate" {
  source = "../modules/acm_certificate"
  providers = {
    aws = aws.us_east_1
  }

  domain  = "*.test.minecraft.paulandsierra.com"
  zone_id = data.aws_route53_zone.prod.zone_id
}

module "networking" {
  source             = "../modules/networking"
  name               = "minecraft-test"
  vpc_cidr           = "10.0.0.0/16"
  public_subnet_cidr = "10.0.1.0/24"
}

module "s3_buckets" {
  source = "../modules/s3_buckets"
  name   = "minecraft-test"
}

module "ec2_role" {
  source        = "../modules/ec2_role"
  name          = "minecraft-test"
  map_bucket    = module.s3_buckets.map_bucket_name
  backup_bucket = module.s3_buckets.backup_bucket_name
}

module "mc_stack" {
  source           = "../modules/mc_stack"
  name             = "minecraft-test"
  ami_id           = data.aws_ami.minecraft.id
  instance_type    = "t3.small"
  vpc_id           = module.networking.vpc_id
  subnet_id        = module.networking.public_subnet_id
  key_name         = "minecraft-packer"
  root_volume_size = 8
  data_volume_size = 32
  ssh_cidr_blocks = [
    "104.230.245.46/32",
  ]
  world_version     = "1.21.8"
  availability_zone = "us-east-2b"

  iam_instance_profile = module.ec2_role.instance_profile_name
  map_bucket           = module.s3_buckets.map_bucket_name
  backup_bucket        = module.s3_buckets.backup_bucket_name
  restic_password      = var.restic_password
}

module "api_lambda" {
  source          = "../modules/api_lambda"
  name            = "minecraft-test"
  instance_id     = module.mc_stack.instance_id
  instance_arn    = module.mc_stack.instance_arn
  dns_name        = "test.${data.aws_route53_zone.prod.name}"
  zone_id         = data.aws_route53_zone.prod.zone_id
  map_bucket_name = module.s3_buckets.map_bucket_name
  cors_origin     = "*"
}

# Manager app (www.test.*)
module "web_ui_www" {
  source = "../modules/web_ui"
  name   = "minecraft-test-www"

  api_endpoint              = module.api_lambda.api_endpoint
  webapp_bucket_name        = module.s3_buckets.webapp_bucket_name
  webapp_bucket_domain_name = module.s3_buckets.webapp_bucket_domain_name

  custom_domain       = "www.test.minecraft.paulandsierra.com"
  acm_certificate_arn = module.acm_certificate.certificate_arn
  zone_id             = data.aws_route53_zone.prod.zone_id

  include_maps  = false
  geo_whitelist = ["US", "CA", "MX"]
}

# Worlds app (maps.test.*)
module "web_ui_maps" {
  source = "../modules/web_ui"
  name   = "minecraft-test-maps"

  api_endpoint              = module.api_lambda.api_endpoint
  webapp_bucket_name        = module.s3_buckets.webapp_maps_bucket_name
  webapp_bucket_domain_name = module.s3_buckets.webapp_maps_bucket_domain_name
  map_bucket_name           = module.s3_buckets.map_bucket_name
  map_bucket_domain_name    = module.s3_buckets.map_bucket_domain_name

  custom_domain       = "maps.test.minecraft.paulandsierra.com"
  acm_certificate_arn = module.acm_certificate.certificate_arn
  zone_id             = data.aws_route53_zone.prod.zone_id

  include_maps  = true
  geo_whitelist = ["US", "CA", "MX"]
}

module "dns_records" {
  source = "../modules/dns_records"

  zone_id  = data.aws_route53_zone.prod.zone_id
  dns_name = "test.${data.aws_route53_zone.prod.name}"
  # ipv4_addresses = module.mc_stack.public_ip != "" ? [module.mc_stack.public_ip] : null
  # ipv6_addresses = module.mc_stack.ipv6_addresses
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
