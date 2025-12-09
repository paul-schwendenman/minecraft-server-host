resource "aws_iam_role" "minecraft_ec2" {
  name = "${var.name}-ec2-role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17",
    Statement = [
      {
        Effect    = "Allow",
        Principal = { Service = "ec2.amazonaws.com" },
        Action    = "sts:AssumeRole"
      }
    ]
  })
}

resource "aws_iam_role_policy" "minecraft_ec2_policy" {
  name = "${var.name}-map-backup-policy"
  role = aws_iam_role.minecraft_ec2.id

  policy = jsonencode({
    Version = "2012-10-17",
    Statement = [
      {
        Sid    = "MapUpload",
        Effect = "Allow",
        Action = ["s3:PutObject", "s3:ListBucket"],
        Resource = [
          "arn:aws:s3:::${var.map_bucket}",
          "arn:aws:s3:::${var.map_bucket}/*"
        ]
      },
      {
        Sid    = "Backups",
        Effect = "Allow",
        Action = ["s3:PutObject", "s3:GetObject", "s3:ListBucket"],
        Resource = [
          "arn:aws:s3:::${var.backup_bucket}",
          "arn:aws:s3:::${var.backup_bucket}/*"
        ]
      }
    ]
  })
}

resource "aws_iam_instance_profile" "minecraft_profile" {
  name = "${var.name}-instance-profile"
  role = aws_iam_role.minecraft_ec2.name
}
