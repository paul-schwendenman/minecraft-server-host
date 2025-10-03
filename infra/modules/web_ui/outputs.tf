output "webapp_url" {
  value = var.dns_name != "" ? "https://${var.dns_name}" : aws_cloudfront_distribution.webapp.domain_name
}
