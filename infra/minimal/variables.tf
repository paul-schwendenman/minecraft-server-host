variable "aws_region" {
  description = "AWS region to deploy into"
  type        = string
  default     = "us-east-2"
}

variable "aws_profile" {
  description = "AWS CLI profile to use"
  type        = string
  default     = "minecraft"
}

variable "instance_type" {
  description = "EC2 instance type"
  type        = string
  default     = "t3a.medium"
}

variable "key_name" {
  description = "Name of the EC2 keypair for SSH access"
  type        = string
}

variable "subnet_id" {
  description = "Subnet to launch into"
  type        = string
}

variable "vpc_id" {
  description = "VPC to associate security group with"
  type        = string
}

variable "world_name" {
  description = "Name of the default world"
  default     = "default"
}

variable "world_version" {
  description = "Minecraft Server Version for default world"
  default     = "1.21.8"
}

variable "world_seed" {
  description = "Minecraft Server Seed"
  default     = ""
}
