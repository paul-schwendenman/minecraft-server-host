terraform {
  required_providers {
    archive = {
      source = "hashicorp/archive"
      version = "~> 2.0"
    }
    aws = {
      source = "hashicorp/aws"
      version = "~> 3.0"
    }
    local = {
      source = "hashicorp/local"
      version = "~> 2.0"
    }
    null = {
      source = "hashicorp/null"
      version = "~> 3.0"
    }
    tls = {
      source = "hashicorp/tls"
      version = "~> 3.0"
    }
  }
}

provider "aws" {
  region  = var.region
}

provider "aws" {
  region = "us-east-1"
  alias  = "virgina"
}

provider "tls" {
}

provider "local" {
}

provider "null" {
}

provider "archive" {
}

locals {
  private_key_filename = "${var.key_name}.pem"
}
