data "archive_file" "api_code" {
  type        = "zip"
  source_file = "${path.module}/api/lambda_handler.py"
  output_path = "${path.module}/api.zip"
}

resource "aws_s3_bucket" "lambda_code" {
#   acl = "private"

  tags = {
    Name = "Minecraft Server - Lambda code"
  }
}

resource "null_resource" "zip_file_upload" {
  triggers = {
      zip_file = data.archive_file.api_code.output_sha
  }
  provisioner "local-exec" {
    command = "aws s3 cp ${data.archive_file.api_code.output_path} s3://${aws_s3_bucket.lambda_code.bucket}/v${var.app_version}/api.zip"
  }
}

resource "aws_lambda_function" "minecraft_api" {
  function_name = "MinecraftAPI"
  depends_on = [null_resource.zip_file_upload]

  s3_bucket = aws_s3_bucket.lambda_code.bucket
  s3_key    = "v${var.app_version}/api.zip"

  # "main" is the filename within the zip file (main.js) and "handler"
  # is the name of the property under which the handler function was
  # exported in that file.
  handler = "lambda_handler.main_handler"
  runtime = "python3.8"

  role = aws_iam_role.lambda_exec.arn
  environment {
    variables = {
      INSTANCE_ID = "${aws_instance.minecraft_server.id}"
      DNS_NAME = "${var.dns_name}"
    }
  }
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

resource "aws_iam_policy" "minecraft_lambda_policy" {
  name   = "minecraft_lambda_policy"
  path   = "/"
  # policy = data.aws_iam_policy_document.lambda.json
  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents"
      ],
      "Resource": "arn:aws:logs:*:*:*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "ec2:StartInstances",
        "ec2:StopInstances"
      ],
      "Resource": "${aws_instance.minecraft_server.arn}"
    },
    {
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeInstances"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "route53:ListHostedZones"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "route53:ListResourceRecordSets",
        "route53:ChangeResourceRecordSets"
      ],
      "Resource": "arn:aws:route53:::hostedzone/${aws_route53_zone.primary.zone_id}"
    }
  ]
}
EOF
}

resource "aws_iam_role_policy_attachment" "test-attach" {
  role       = aws_iam_role.lambda_exec.name
  policy_arn = aws_iam_policy.minecraft_lambda_policy.arn
}

resource "aws_api_gateway_rest_api" "minecraft_gateway" {
  name        = "MinecraftAPI"
  description = "Minecraft Management Application"
}

resource "aws_api_gateway_resource" "proxy" {
  rest_api_id = aws_api_gateway_rest_api.minecraft_gateway.id
  parent_id   = aws_api_gateway_rest_api.minecraft_gateway.root_resource_id
  path_part   = "{proxy+}"
}

resource "aws_api_gateway_method" "proxy" {
  rest_api_id   = aws_api_gateway_rest_api.minecraft_gateway.id
  resource_id   = aws_api_gateway_resource.proxy.id
  http_method   = "ANY"
  authorization = "NONE"
}

resource "aws_api_gateway_integration" "lambda" {
  rest_api_id = aws_api_gateway_rest_api.minecraft_gateway.id
  resource_id = aws_api_gateway_method.proxy.resource_id
  http_method = aws_api_gateway_method.proxy.http_method

  integration_http_method = "POST"
  type                    = "AWS_PROXY"
  uri                     = aws_lambda_function.minecraft_api.invoke_arn
}

resource "aws_api_gateway_method" "proxy_root" {
  rest_api_id   = aws_api_gateway_rest_api.minecraft_gateway.id
  resource_id   = aws_api_gateway_rest_api.minecraft_gateway.root_resource_id
  http_method   = "ANY"
  authorization = "NONE"
}

resource "aws_api_gateway_integration" "lambda_root" {
  rest_api_id = aws_api_gateway_rest_api.minecraft_gateway.id
  resource_id = aws_api_gateway_method.proxy_root.resource_id
  http_method = aws_api_gateway_method.proxy_root.http_method

  integration_http_method = "POST"
  type                    = "AWS_PROXY"
  uri                     = aws_lambda_function.minecraft_api.invoke_arn
}

resource "aws_api_gateway_deployment" "dev" {
  depends_on = [
    aws_api_gateway_integration.lambda,
    aws_api_gateway_integration.lambda_root,
  ]

  rest_api_id = aws_api_gateway_rest_api.minecraft_gateway.id
  stage_name  = "dev"
}

resource "aws_lambda_permission" "apigw" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.minecraft_api.function_name
  principal     = "apigateway.amazonaws.com"

  # The "/*/*" portion grants access from any method on any resource
  # within the API Gateway REST API.
  source_arn = "${aws_api_gateway_rest_api.minecraft_gateway.execution_arn}/*/*"
}

output "bucket" {
  value = "${aws_s3_bucket.lambda_code.bucket}"
}

output "base_url" {
  value = aws_api_gateway_deployment.dev.invoke_url
}
