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

resource "aws_s3_bucket_lifecycle_configuration" "backups" {
  bucket = aws_s3_bucket.backups.id

  rule {
    id     = "expire-old-backups"
    status = "Enabled"

    # Applies to all objects
    filter {}

    # Move objects to Glacier after 30 days
    transition {
      days          = 30
      storage_class = "GLACIER"
    }

    # Permanently delete after 90 days
    expiration {
      days = 90
    }
  }
}
