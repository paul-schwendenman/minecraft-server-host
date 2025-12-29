variable "aws_profile" {
  description = "AWS CLI profile to use"
  type        = string
  default     = "minecraft"
}

variable "github_repo" {
  description = "GitHub repository in format 'org/repo'"
  type        = string
}

variable "lambda_prefixes" {
  description = "List of Lambda function prefixes for deployment permissions"
  type        = list(string)
  default     = ["minecraft-test", "minecraft-prod"]
}
