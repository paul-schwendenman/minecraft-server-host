resource "aws_s3_bucket" "www" {
  // Our bucket's name is going to be the same as our site's domain name.
  bucket = var.webapp_dns_name
  // Because we want our site to be available on the internet, we set this so
  // anyone can read this bucket.
  acl = "public-read"
  // We also need to create a policy that allows anyone to view the content.
  // This is basically duplicating what we did in the ACL but it's required by
  // AWS. This post: http://amzn.to/2Fa04ul explains why.
  policy = <<POLICY
{
  "Version":"2012-10-17",
  "Statement":[
    {
      "Sid":"AddPerm",
      "Effect":"Allow",
      "Principal": "*",
      "Action":["s3:GetObject"],
      "Resource":["arn:aws:s3:::${var.webapp_dns_name}/*"]
    }
  ]
}
POLICY

  // S3 understands what it means to host a website.
  website {
    // Here we tell S3 what to use when a request comes in to the root
    // ex. https://www.runatlantis.io
    index_document = "index.html"
    // The page to serve up if a request results in an error or a non-existing
    // page.
    error_document = "404.html"
  }
}

// Use the AWS Certificate Manager to create an SSL cert for our domain.
// This resource won't be created until you receive the email verifying you
// own the domain and you click on the confirmation link.
resource "aws_acm_certificate" "certificate" {
  // We want a wildcard cert so we can host subdomains later.
  domain_name       = var.webapp_dns_name
  validation_method = "DNS"
  provider          = aws.virgina

  // We also want the cert to be valid for the root domain even though we'll be
  // redirecting to the www. domain immediately.
  subject_alternative_names = ["${var.dns_name}"]
}

resource "aws_cloudfront_distribution" "www_distribution" {
  // origin is where CloudFront gets its content from.
  origin {
    // We need to set up a "custom" origin because otherwise CloudFront won't
    // redirect traffic from the root domain to the www domain, that is from
    // runatlantis.io to www.runatlantis.io.
    custom_origin_config {
      // These are all the defaults.
      http_port              = "80"
      https_port             = "443"
      origin_protocol_policy = "http-only"
      origin_ssl_protocols   = ["TLSv1", "TLSv1.1", "TLSv1.2"]
    }

    // Here we're using our S3 bucket's URL!
    domain_name = aws_s3_bucket.www.website_endpoint
    // This can be any name to identify this origin.
    origin_id = var.webapp_dns_name
  }

  enabled             = true
  default_root_object = "index.html"

  // All values are defaults from the AWS console.
  default_cache_behavior {
    viewer_protocol_policy = "redirect-to-https"
    compress               = true
    allowed_methods        = ["GET", "HEAD"]
    cached_methods         = ["GET", "HEAD"]
    // This needs to match the `origin_id` above.
    target_origin_id = var.webapp_dns_name
    min_ttl          = 0
    default_ttl      = 86400
    max_ttl          = 31536000

    forwarded_values {
      query_string = false
      cookies {
        forward = "none"
      }
    }
  }

  // Here we're ensuring we can hit this distribution using www.runatlantis.io
  // rather than the domain name CloudFront gives us.
  aliases = ["${var.webapp_dns_name}"]

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  // Here's where our certificate is loaded in!
  viewer_certificate {
    acm_certificate_arn = aws_acm_certificate.certificate.arn
    ssl_support_method  = "sni-only"
  }
}

resource "aws_route53_record" "cert_validation" {
  name    = aws_acm_certificate.certificate.domain_validation_options.0.resource_record_name
  type    = aws_acm_certificate.certificate.domain_validation_options.0.resource_record_type
  zone_id = aws_route53_zone.primary.zone_id
  records = ["${aws_acm_certificate.certificate.domain_validation_options.0.resource_record_value}"]
  ttl     = 60
}

resource "aws_route53_record" "cert_validation_alt1" {
  name    = aws_acm_certificate.certificate.domain_validation_options.1.resource_record_name
  type    = aws_acm_certificate.certificate.domain_validation_options.1.resource_record_type
  zone_id = aws_route53_zone.primary.zone_id
  records = ["${aws_acm_certificate.certificate.domain_validation_options.1.resource_record_value}"]
  ttl     = 60
}

data "archive_file" "ui_code" {
  type        = "zip"
  source_dir  = "${path.module}/ui/public/"
  output_path = "${path.module}/ui.zip"
}

data "local_file" "ui_dependencies" {
  filename = "${path.module}/ui/package-lock.json"
}

resource "null_resource" "ui_dependencies" {
  triggers = {
    package_lock = sha1(data.local_file.ui_dependencies.content)
  }

  provisioner "local-exec" {
    command     = "npm install"
    working_dir = "ui"
  }
}

resource "local_file" "rollup_config" {
  content         = templatefile("ui/rollup.config.js.tmpl", { api_url : aws_api_gateway_deployment.dev.invoke_url })
  filename        = "ui/rollup.config.js"
  file_permission = "0644"
}

resource "null_resource" "ui_build" {
  depends_on = [null_resource.ui_dependencies, local_file.rollup_config]

  provisioner "local-exec" {
    command     = "npm run build"
    working_dir = "ui"
  }
}

resource "null_resource" "webapp_upload" {
  triggers = {
    zip_file = data.archive_file.ui_code.output_sha
    build    = null_resource.ui_build.id
  }
  depends_on = [null_resource.ui_build]

  provisioner "local-exec" {
    command     = "aws s3 sync public/ s3://${aws_s3_bucket.www.bucket}"
    working_dir = "ui"
  }
}
