variable "name" {
  description = "Prefix for webapp resources"
  type        = string
}

variable "api_endpoint" {
  description = "The Lambda API endpoint URL"
  type        = string
}

variable "dns_name" {
  description = "Optional DNS name for the webapp"
  type        = string
  default     = ""
}

variable "zone_id" {
  description = "Optional Route53 hosted zone ID if dns_name set"
  type        = string
  default     = ""
}

variable "geo_whitelist" {
  description = "Optional list of ISO country codes allowed (e.g., [\"US\", \"CA\", \"MX\"]). Empty = unrestricted"
  type        = list(string)
  default     = []
}

variable "webapp_bucket_name" {
  description = "The name of the S3 bucket used for the web UI"
  type        = string
}

variable "webapp_bucket_domain_name" {
  description = "The regional domain name of the webapp S3 bucket"
  type        = string
}

variable "map_bucket_name" {
  description = "Name of the S3 bucket that stores rendered maps"
  type        = string
}
