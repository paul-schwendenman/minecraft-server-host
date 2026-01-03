variable "name" {
  description = "Prefix for role and instance profile"
  type        = string
}

variable "map_bucket" {
  description = "Name of the S3 bucket used for map uploads"
  type        = string
}

variable "backup_bucket" {
  description = "Name of the S3 bucket used for backups"
  type        = string
}

variable "route53_zone_id" {
  description = "Route53 hosted zone ID for SSHFP record updates (optional)"
  type        = string
  default     = ""
}
