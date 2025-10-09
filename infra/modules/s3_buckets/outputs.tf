output "map_bucket_name" {
  value = aws_s3_bucket.maps.bucket
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
