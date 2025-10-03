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
