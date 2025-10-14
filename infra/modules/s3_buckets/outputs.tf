output "map_bucket_name" {
  value = aws_s3_bucket.maps.bucket
}

output "map_bucket_domain_name" {
  value = aws_s3_bucket.maps.bucket_regional_domain_name
}

output "map_bucket_s3_website" {
  value = aws_s3_bucket_website_configuration.maps.website_endpoint
}

output "backup_bucket_name" {
  value = aws_s3_bucket.backups.bucket
}

output "webapp_bucket_name" {
  value = aws_s3_bucket.webapp.bucket
}

output "webapp_bucket_domain_name" {
  value = aws_s3_bucket.webapp.bucket_regional_domain_name
}
