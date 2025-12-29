output "webapp_url" {
  value = var.custom_domain != "" ? "https://${var.custom_domain}" : (var.dns_name != "" ? "https://${var.dns_name}" : "https://${aws_cloudfront_distribution.webapp.domain_name}")
}

output "distribution_id" {
  description = "CloudFront distribution ID for cache invalidation"
  value       = aws_cloudfront_distribution.webapp.id
}

output "distribution_arn" {
  description = "CloudFront distribution ARN"
  value       = aws_cloudfront_distribution.webapp.arn
}
