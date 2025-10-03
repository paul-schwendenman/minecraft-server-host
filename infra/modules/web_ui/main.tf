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

resource "aws_cloudfront_distribution" "webapp" {
  enabled             = true
  default_root_object = "index.html"

  origin {
    domain_name = aws_s3_bucket.webapp.bucket_regional_domain_name
    origin_id   = "webapp-origin"
  }

  default_cache_behavior {
    target_origin_id       = "webapp-origin"
    viewer_protocol_policy = "redirect-to-https"
    allowed_methods        = ["GET", "HEAD"]
    cached_methods         = ["GET", "HEAD"]

    forwarded_values {
      query_string = false
      cookies {
        forward = "none"
      }
    }
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }
}

resource "aws_route53_record" "webapp_dns" {
  count   = var.dns_name != "" && var.zone_id != "" ? 1 : 0
  name    = var.dns_name
  type    = "CNAME"
  zone_id = var.zone_id
  ttl     = 300
  records = [aws_cloudfront_distribution.webapp.domain_name]
}
