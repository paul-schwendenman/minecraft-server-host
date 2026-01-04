variable "name" {
  description = "Name prefix for resources"
  type        = string
}

variable "ami_id" {
  description = "AMI to use for Minecraft server"
  type        = string
}

variable "availability_zone" {
  description = "Availability Zone where the server and its world volume should be placed"
  type        = string
}

variable "instance_type" {
  description = "EC2 instance type"
  type        = string
  default     = "t3.small"
}

variable "vpc_id" {
  description = "VPC where server runs"
  type        = string
}

variable "subnet_id" {
  description = "Subnet for instance"
  type        = string
}

variable "key_name" {
  description = "EC2 key pair for SSH"
  type        = string
}

variable "root_volume_size" {
  description = "Size of root EBS volume (GB)"
  type        = number
  default     = 8
}

variable "data_volume_size" {
  description = "Size of Minecraft world EBS volume (GB)"
  type        = number
  default     = 8
}

variable "data_volume_device_name" {
  description = "Device name to attach EBS volume as (Linux)"
  type        = string
  default     = "/dev/sdf"
}

variable "ssh_cidr_blocks" {
  description = "CIDRs allowed to SSH"
  type        = list(string)
  default     = ["0.0.0.0/0"]
}

variable "iam_instance_profile" {
  description = "IAM instance profile to attach to the EC2 instance"
  type        = string
}

variable "map_bucket" {
  description = "Name of the S3 bucket used for maps"
  type        = string
}

variable "backup_bucket" {
  description = "S3 bucket for world backups"
  type        = string
  default     = ""
}

variable "restic_password" {
  description = "Password for restic backup encryption"
  type        = string
  sensitive   = true
}

variable "world_name" {
  description = "Minecraft world name"
  type        = string
  default     = "default"
}

variable "world_version" {
  description = "Minecraft server version"
  type        = string
}

variable "world_seed" {
  description = "Seed for Minecraft world"
  type        = string
  default     = ""
}

variable "route53_zone_id" {
  description = "Route53 zone ID for publishing SSHFP records (optional)"
  type        = string
  default     = ""
}

variable "route53_dns_name" {
  description = "DNS name for SSHFP records (optional, requires route53_zone_id)"
  type        = string
  default     = ""
}
