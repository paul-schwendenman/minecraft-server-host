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
