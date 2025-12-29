output "map_bucket_name" {
  value = aws_s3_bucket.maps.bucket
}

output "map_bucket_domain_name" {
  value = aws_s3_bucket.maps.bucket_regional_domain_name
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

output "webapp_maps_bucket_name" {
  value = aws_s3_bucket.webapp_maps.bucket
}

output "webapp_maps_bucket_domain_name" {
  value = aws_s3_bucket.webapp_maps.bucket_regional_domain_name
}
