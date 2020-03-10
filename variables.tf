variable "region" {
  description = "region for minecraft server"
  default = "us-east-2"
}

variable "instance_type" {
  description = "AWS instance type for the minecraft server"
  default = "t3.small"
}

variable "security_group" {
  description = "name for the AWS security group"
  default = "minecraft"
}

variable "key_name" {
  description = "name for generated ssh key"
  default = "minecraft"
}

variable "app_version" {
}

variable "dns_name" {
}
