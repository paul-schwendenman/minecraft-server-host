variable "aws_profile" {
  description = "AWS CLI profile to use"
  type        = string
  default     = "minecraft"
}

provider "aws" {
  region  = "us-east-2"
  profile = var.aws_profile
}

# Get outputs from test environment
data "terraform_remote_state" "test" {
  backend = "local"
  config = {
    path = "../test/terraform.tfstate"
  }
}

# Get outputs from prod environment
data "terraform_remote_state" "prod" {
  backend = "local"
  config = {
    path = "../prod/terraform.tfstate"
  }
}

module "github_actions_role" {
  source = "../modules/github_actions_role"

  github_repo = "paul-schwendenman/minecraft-server-host"

  # S3 buckets from both environments
  s3_buckets = [
    # Test
    data.terraform_remote_state.test.outputs.webapp_bucket_name,
    data.terraform_remote_state.test.outputs.webapp_maps_bucket_name,
    # Prod
    data.terraform_remote_state.prod.outputs.webapp_bucket_name,
    data.terraform_remote_state.prod.outputs.webapp_maps_bucket_name,
  ]

  # CloudFront distributions from both environments
  cloudfront_distribution_arns = [
    # Test
    "arn:aws:cloudfront::${data.aws_caller_identity.current.account_id}:distribution/${data.terraform_remote_state.test.outputs.cloudfront_www_distribution_id}",
    "arn:aws:cloudfront::${data.aws_caller_identity.current.account_id}:distribution/${data.terraform_remote_state.test.outputs.cloudfront_maps_distribution_id}",
    # Prod
    "arn:aws:cloudfront::${data.aws_caller_identity.current.account_id}:distribution/${data.terraform_remote_state.prod.outputs.cloudfront_www_distribution_id}",
    "arn:aws:cloudfront::${data.aws_caller_identity.current.account_id}:distribution/${data.terraform_remote_state.prod.outputs.cloudfront_maps_distribution_id}",
  ]

  # Lambda prefixes for both environments
  lambda_prefixes = [
    "minecraft-test",
    "minecraft-prod",
  ]
}

data "aws_caller_identity" "current" {}

output "github_actions_role_arn" {
  description = "Add this to GitHub Secrets as AWS_ROLE_ARN"
  value       = module.github_actions_role.role_arn
}
