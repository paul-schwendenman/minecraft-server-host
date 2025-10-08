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

resource "aws_cloudfront_origin_access_control" "webapp" {
  name                              = "${var.name}-oac"
  description                       = "Access control for ${var.name} CloudFront to S3"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

resource "aws_cloudfront_distribution" "webapp" {
  enabled             = true
  default_root_object = "index.html"

  # Origin 1: S3
  origin {
    domain_name = aws_s3_bucket.webapp.bucket_regional_domain_name
    origin_id   = "webapp-origin"

    origin_access_control_id = aws_cloudfront_origin_access_control.webapp.id
  }

  # Origin 2: API Gateway
  origin {
    domain_name = replace(replace(var.api_endpoint, "https://", ""), "http://", "")
    origin_id   = "api-origin"

    custom_origin_config {
      origin_protocol_policy = "https-only"
      http_port              = 80
      https_port             = 443
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  # Origin 3: Maps S3 Bucket
  origin {
    domain_name = "${var.map_bucket_name}.s3.amazonaws.com"
    origin_id   = "maps-origin"

    origin_access_control_id = aws_cloudfront_origin_access_control.webapp.id
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

  # Cache behavior for API calls
  ordered_cache_behavior {
    path_pattern           = "/api/*"
    target_origin_id       = "api-origin"
    viewer_protocol_policy = "redirect-to-https"
    allowed_methods        = ["GET", "HEAD", "OPTIONS", "PUT", "POST", "PATCH", "DELETE"]
    cached_methods         = ["GET", "HEAD"]

    forwarded_values {
      query_string = true
      headers      = ["Authorization"]
      cookies {
        forward = "all"
      }
    }
  }

  # Maps static content: /maps/<world>/<dimension>/*
  ordered_cache_behavior {
    path_pattern           = "/maps/*/*"
    target_origin_id       = "maps-origin"
    viewer_protocol_policy = "redirect-to-https"
    allowed_methods        = ["GET", "HEAD", "OPTIONS"]
    cached_methods         = ["GET", "HEAD"]
    compress               = true

    forwarded_values {
      query_string = false
      cookies {
        forward = "none"
      }
    }
  }

  # Maps SPA routes: /maps and /maps/<world>
  # Still handled by the Svelte UI (so routing works client-side)
  ordered_cache_behavior {
    path_pattern           = "/maps*"
    target_origin_id       = "webapp-origin"
    viewer_protocol_policy = "redirect-to-https"
    allowed_methods        = ["GET", "HEAD", "OPTIONS"]
    cached_methods         = ["GET", "HEAD"]
    compress               = true

    forwarded_values {
      query_string = false
      cookies {
        forward = "none"
      }
    }
  }


  restrictions {
    geo_restriction {
      restriction_type = length(var.geo_whitelist) > 0 ? "whitelist" : "none"
      locations        = var.geo_whitelist
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  custom_error_response {
    error_code         = 404
    response_code      = 200
    response_page_path = "/index.html"
  }

  custom_error_response {
    error_code         = 403
    response_code      = 200
    response_page_path = "/index.html"
  }

}

resource "aws_s3_bucket_policy" "webapp" {
  bucket = aws_s3_bucket.webapp.id

  policy = jsonencode({
    Version = "2012-10-17",
    Statement = [
      {
        Sid    = "AllowCloudFrontServicePrincipalRead",
        Effect = "Allow",
        Principal = {
          Service = "cloudfront.amazonaws.com"
        },
        Action   = ["s3:GetObject"],
        Resource = "${aws_s3_bucket.webapp.arn}/*",
        Condition = {
          StringEquals = {
            "AWS:SourceArn" = aws_cloudfront_distribution.webapp.arn
          }
        }
      }
    ]
  })
}

resource "aws_s3_bucket_policy" "maps" {
  bucket = var.map_bucket_name

  policy = jsonencode({
    Version = "2012-10-17",
    Statement = [
      {
        Sid    = "AllowCloudFrontReadMaps",
        Effect = "Allow",
        Principal = {
          Service = "cloudfront.amazonaws.com"
        },
        Action   = ["s3:GetObject"],
        Resource = "arn:aws:s3:::${var.map_bucket_name}/*",
        Condition = {
          StringEquals = {
            "AWS:SourceArn" = aws_cloudfront_distribution.webapp.arn
          }
        }
      }
    ]
  })
}


resource "aws_route53_record" "webapp_dns" {
  count   = var.dns_name != "" && var.zone_id != "" ? 1 : 0
  name    = var.dns_name
  type    = "CNAME"
  zone_id = var.zone_id
  ttl     = 300
  records = [aws_cloudfront_distribution.webapp.domain_name]
}
