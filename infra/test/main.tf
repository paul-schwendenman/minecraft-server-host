variable "aws_profile" {
  description = "AWS CLI profile to use"
  type        = string
  default     = "minecraft"
}

provider "aws" {
  region  = "us-east-2"
  profile = var.aws_profile
}

data "aws_route53_zone" "prod" {
  name         = "minecraft.paulandsierra.com."
  private_zone = false
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
  ami_id           = var.ami_id
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
}

module "api_lambda" {
  source          = "../modules/api_lambda"
  name            = "minecraft-test"
  instance_id     = module.mc_stack.instance_id
  instance_arn    = module.mc_stack.instance_arn
  dns_name        = "testmc.${data.aws_route53_zone.prod.name}"
  zone_id         = data.aws_route53_zone.prod.zone_id
  map_bucket_name = module.s3_buckets.map_bucket_name
  cors_origin     = "*"
}

module "web_ui" {
  source = "../modules/web_ui"
  name   = "minecraft-test"

  api_endpoint              = module.api_lambda.api_endpoint
  webapp_bucket_name        = module.s3_buckets.webapp_bucket_name
  webapp_bucket_domain_name = module.s3_buckets.webapp_bucket_domain_name
  map_bucket_name           = module.s3_buckets.map_bucket_name
  map_bucket_domain_name    = module.s3_buckets.map_bucket_domain_name

  # optional DNS if you want a pretty domain
  # dns_name = "testui.minecraft.paulandsierra.com"
  # zone_id  = data.aws_route53_zone.prod.zone_id

  # Restrict to North America for testing
  geo_whitelist = ["US", "CA", "MX"]
}

module "dns_records" {
  source = "../modules/dns_records"

  zone_id  = data.aws_route53_zone.prod.zone_id
  dns_name = "testmc.${data.aws_route53_zone.prod.name}"
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

output "webapp_url" {
  value = module.web_ui.webapp_url
}

output "map_bucket_name" {
  value = module.s3_buckets.map_bucket_name
}
