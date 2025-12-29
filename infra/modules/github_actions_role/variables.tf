variable "github_repo" {
  description = "GitHub repository in format 'org/repo'"
  type        = string
}

variable "role_name" {
  description = "Name of the IAM role"
  type        = string
  default     = "GitHubActionsRole"
}

variable "s3_buckets" {
  description = "List of S3 bucket names GitHub Actions can deploy to"
  type        = list(string)
  default     = []
}

variable "cloudfront_distribution_arns" {
  description = "List of CloudFront distribution ARNs GitHub Actions can invalidate"
  type        = list(string)
  default     = []
}

variable "lambda_prefixes" {
  description = "List of Lambda function prefixes (e.g., ['minecraft-test', 'minecraft-prod'])"
  type        = list(string)
  default     = []
}

variable "include_packer_permissions" {
  description = "Include EC2/AMI permissions for Packer builds"
  type        = bool
  default     = true
}
