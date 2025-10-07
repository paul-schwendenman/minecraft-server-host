output "map_bucket_name" {
  value = aws_s3_bucket.maps.bucket
}

output "backup_bucket_name" {
  value = aws_s3_bucket.backups.bucket
}
