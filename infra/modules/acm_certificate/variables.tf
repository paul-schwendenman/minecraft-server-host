variable "domain" {
  description = "Domain name for the certificate (e.g., *.example.com for wildcard)"
  type        = string
}

variable "zone_id" {
  description = "Route53 hosted zone ID for DNS validation"
  type        = string
}
