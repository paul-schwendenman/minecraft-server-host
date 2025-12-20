variable "zone_id" {
  description = "The Route53 hosted zone ID for the domain."
  type        = string
}

variable "dns_name" {
  description = "Fully qualified DNS name for the record (e.g., testmc.example.com)."
  type        = string
}

variable "ipv4_addresses" {
  description = "IPv4 addresses for the A record. Optional."
  type        = list(string)
  default     = []
}

variable "ipv6_addresses" {
  description = "IPv6 addresses for the AAAA record. Optional."
  type        = list(string)
  default     = []
}

variable "create_a_record" {
  description = "Whether to create the A record. Set to true after EC2 instance exists."
  type        = bool
  default     = false
}

variable "create_aaaa_record" {
  description = "Whether to create the AAAA record. Set to true after EC2 instance exists."
  type        = bool
  default     = false
}

variable "ttl" {
  description = "TTL for DNS records."
  type        = number
  default     = 300
}
