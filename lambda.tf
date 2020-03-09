data "archive_file" "api_code" {
  type        = "zip"
  source_file = "${path.module}/api/main.js"
  output_path = "${path.module}/api.zip"
}

resource "aws_s3_bucket" "lambda_code" {
#   acl = "private"

  tags = {
    Name = "Minecraft Server - Lambda code"
  }

  provisioner "local-exec" {
    command = "aws s3 cp ${data.archive_file.api_code.output_path} s3://${aws_s3_bucket.lambda_code.bucket}/v1.0.0/api.zip"
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
