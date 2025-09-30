variable "name" {
  description = "Name prefix for resources"
  type        = string
}

variable "ami_id" {
  description = "AMI to use for Minecraft server"
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

variable "ssh_cidr_blocks" {
  description = "CIDRs allowed to SSH"
  type        = list(string)
  default     = ["0.0.0.0/0"]
}
