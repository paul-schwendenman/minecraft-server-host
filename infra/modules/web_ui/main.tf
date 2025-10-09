###############################################
# CloudFront + S3 Setup for Web UI + Maps
###############################################

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

  ##########################################
  # ORIGINS
  ##########################################

  # Origin 1: Webapp (Svelte UI)
  origin {
    domain_name = var.webapp_bucket_domain_name
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

  # Origin 3: Maps bucket (uNmINeD exports)
  origin {
    domain_name = var.map_bucket_domain_name
    origin_id   = "maps-origin"

    origin_access_control_id = aws_cloudfront_origin_access_control.webapp.id
  }

  ##########################################
  # CACHE BEHAVIORS
  ##########################################

  # 1. Maps: static map tiles and HTML
  ordered_cache_behavior {
    path_pattern           = "/maps/*"
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

  # 2. API calls
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
  ordered_cache_behavior {
    path_pattern           = "/api/*/*"
    target_origin_id       = "api-origin"
    viewer_protocol_policy = "redirect-to-https"
    allowed_methods        = ["GET", "HEAD", "OPTIONS", "PUT", "POST", "PATCH", "DELETE"]
    cached_methods         = ["GET", "HEAD"]
    compress               = true

    forwarded_values {
      query_string = true
      headers      = ["Authorization"]
      cookies {
        forward = "all"
      }
    }
  }

  # 3. Default (UI SPA)
  default_cache_behavior {
    target_origin_id       = "webapp-origin"
    viewer_protocol_policy = "redirect-to-https"
    allowed_methods        = ["GET", "HEAD"]
    cached_methods         = ["GET", "HEAD"]
    compress               = true

    forwarded_values {
      query_string = false
      cookies {
        forward = "none"
      }
    }
  }

  ##########################################
  # SETTINGS
  ##########################################

  restrictions {
    geo_restriction {
      restriction_type = length(var.geo_whitelist) > 0 ? "whitelist" : "none"
      locations        = var.geo_whitelist
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  # Let SPA handle 403/404 (single-page app routing)
  custom_error_response {
    error_code         = 403
    response_code      = 200
    response_page_path = "/index.html"
  }

  custom_error_response {
    error_code         = 404
    response_code      = 200
    response_page_path = "/index.html"
  }
}

###############################################
# S3 Bucket Policies
###############################################

# Webapp bucket (Svelte UI)
resource "aws_s3_bucket_policy" "webapp" {
  bucket = var.webapp_bucket_name

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
        Resource = "arn:aws:s3:::${var.webapp_bucket_name}/*",
        Condition = {
          StringEquals = {
            "AWS:SourceArn" = aws_cloudfront_distribution.webapp.arn
          }
        }
      }
    ]
  })
}

# Maps bucket (Unmined exports)
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

###############################################
# DNS (optional)
###############################################
resource "aws_route53_record" "webapp_dns" {
  count   = var.dns_name != "" && var.zone_id != "" ? 1 : 0
  name    = var.dns_name
  type    = "CNAME"
  zone_id = var.zone_id
  ttl     = 300
  records = [aws_cloudfront_distribution.webapp.domain_name]
}
