provider "aws" {
  version = "~> 2.0"
  region  = var.region
}

provider "aws" {
  # version = "~> 2.0"
  region  = "us-east-1"
  alias = "virgina"
}

provider "tls" {
  version = "~> 2.1"
}

provider "local" {
  version = "~> 1.4"
}

provider "null" {
  version = "~> 2.1"
}

provider "archive" {
  version = "~> 1.3"
}

locals {
    private_key_filename = "${var.key_name}.pem"
}
