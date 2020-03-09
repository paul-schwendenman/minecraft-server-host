resource "aws_s3_bucket" "lambda_code" {
#   acl = "private"

  tags = {
    Name = "Minecraft Server - Lambda code"
  }

  provisioner "local-exec" {
    command = "aws s3 cp api.zip s3://${aws_s3_bucket.lambda_code.bucket}/v1.0.0/api.zip"
  }
}

resource "aws_lambda_function" "minecraft_api" {
  function_name = "MinecraftAPI"

  s3_bucket = aws_s3_bucket.lambda_code.bucket
  s3_key    = "v1.0.0/api.zip"

  # "main" is the filename within the zip file (main.js) and "handler"
  # is the name of the property under which the handler function was
  # exported in that file.
  handler = "main.handler"
  runtime = "nodejs10.x"

  role = aws_iam_role.lambda_exec.arn
}

resource "aws_iam_role" "lambda_exec" {
  name = "serverless_example_lambda"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

output "bucket" {
  value = "${aws_s3_bucket.lambda_code.bucket}"
}
