variable "name" {
  description = "Prefix for resources"
  type        = string
}

variable "instance_id" {
  description = "Minecraft EC2 instance ID to control"
  type        = string
}

variable "instance_arn" {
  description = "EC2 instance ARN for Minecraft server"
  type        = string
}

variable "dns_name" {
  description = "DNS record name to update (optional)"
  type        = string
  default     = ""
}

variable "cors_origin" {
  description = "CORS origin allowed in responses (optional)"
  type        = string
  default     = "*"
}

variable "zone_id" {
  description = "Optional Route53 hosted zone ID to allow DNS updates against"
  type        = string
  default     = ""
}
