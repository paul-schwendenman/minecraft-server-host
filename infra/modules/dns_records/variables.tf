variable "zone_id" {
  description = "The Route53 hosted zone ID for the domain."
  type        = string
}

variable "dns_name" {
  description = "Fully qualified DNS name for the record (e.g., testmc.example.com)."
  type        = string
}

variable "ipv4_address" {
  description = "Single IPv4 address for the A record. Optional."
  type        = string
  default     = null
}

variable "ipv6_address" {
  description = "Single IPv6 address for the AAAA record. Optional."
  type        = string
  default     = null
}

variable "ttl" {
  description = "TTL for DNS records."
  type        = number
  default     = 300
}
