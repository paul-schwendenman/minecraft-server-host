locals {
  prefix = lower(var.name)
}

# Web app - www (manager app)
resource "aws_s3_bucket" "webapp" {
  bucket        = "${var.name}-webapp"
  force_destroy = true
}

resource "aws_s3_bucket_website_configuration" "webapp" {
  bucket = aws_s3_bucket.webapp.id
  index_document {
    suffix = "index.html"
  }
}

# Web app - maps (worlds app)
resource "aws_s3_bucket" "webapp_maps" {
  bucket        = "${var.name}-webapp-maps"
  force_destroy = true
}

resource "aws_s3_bucket_website_configuration" "webapp_maps" {
  bucket = aws_s3_bucket.webapp_maps.id
  index_document {
    suffix = "index.html"
  }
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

resource "aws_s3_bucket_website_configuration" "maps" {
  bucket = aws_s3_bucket.maps.id

  index_document {
    suffix = "index.html"
  }

  error_document {
    key = "index.html"
  }
}
