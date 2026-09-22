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

# Build artifacts mirror (unmined-cli today; shared by test and prod since
# these are build inputs, not environment data). See docs/plans/unmined-cli-docs-plan.md.
resource "aws_s3_bucket" "artifacts" {
  bucket        = "minecraft-server-host-artifacts"
  force_destroy = false
}

resource "aws_s3_bucket_versioning" "artifacts" {
  bucket = aws_s3_bucket.artifacts.id
  versioning_configuration { status = "Enabled" }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "artifacts" {
  bucket = aws_s3_bucket.artifacts.id
  rule {
    apply_server_side_encryption_by_default { sse_algorithm = "AES256" }
  }
}

module "github_actions_role" {
  source = "../modules/github_actions_role"

  github_repo = var.github_repo

  # S3 buckets from both environments, plus the shared artifacts bucket
  s3_buckets = [
    # Test
    data.terraform_remote_state.test.outputs.webapp_bucket_name,
    data.terraform_remote_state.test.outputs.webapp_maps_bucket_name,
    # Prod
    data.terraform_remote_state.prod.outputs.webapp_bucket_name,
    data.terraform_remote_state.prod.outputs.webapp_maps_bucket_name,
    # Shared
    aws_s3_bucket.artifacts.bucket,
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

  lambda_prefixes = var.lambda_prefixes
}

data "aws_caller_identity" "current" {}

output "github_actions_role_arn" {
  description = "Add this to GitHub Secrets as AWS_ROLE_ARN"
  value       = module.github_actions_role.role_arn
}

output "artifacts_bucket_name" {
  description = "S3 bucket that mirrors external build inputs (unmined-cli, ...)"
  value       = aws_s3_bucket.artifacts.bucket
}
