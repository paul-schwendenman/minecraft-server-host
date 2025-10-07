locals {
  prefix = lower(var.name)
}

# World backups
resource "aws_s3_bucket" "backups" {
  bucket        = "${local.prefix}-backups"
  force_destroy = true
}

resource "aws_s3_bucket_versioning" "backups" {
  bucket = aws_s3_bucket.backups.id
  versioning_configuration { status = "Enabled" }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "backups" {
  bucket = aws_s3_bucket.backups.id
  rule {
    apply_server_side_encryption_by_default { sse_algorithm = "AES256" }
  }
}

# Rendered world maps
resource "aws_s3_bucket" "maps" {
  bucket        = "${local.prefix}-maps"
  force_destroy = true
}

resource "aws_s3_bucket_versioning" "maps" {
  bucket = aws_s3_bucket.maps.id
  versioning_configuration { status = "Enabled" }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "maps" {
  bucket = aws_s3_bucket.maps.id
  rule {
    apply_server_side_encryption_by_default { sse_algorithm = "AES256" }
  }
}

# Optional lifecycle for backups older than 90 days
resource "aws_s3_bucket_lifecycle_configuration" "backups" {
  bucket = aws_s3_bucket.backups.id

  rule {
    id     = "expire-old-backups"
    status = "Enabled"

    filter {}  # ← apply to all objects

    expiration {
      days = 90
    }
  }
}
