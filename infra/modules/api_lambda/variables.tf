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
