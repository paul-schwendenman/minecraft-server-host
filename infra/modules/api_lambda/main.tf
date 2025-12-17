resource "aws_iam_role" "lambda_exec" {
  name = "${var.name}-lambda-exec"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "lambda.amazonaws.com"
        }
      }
    ]
  })
}

resource "aws_iam_role_policy" "lambda_policy" {
  name = "${var.name}-lambda-policy"
  role = aws_iam_role.lambda_exec.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = concat(
      [
        {
          Effect = "Allow"
          Action = [
            "ec2:StartInstances",
            "ec2:StopInstances"
          ]
          Resource = var.instance_arn
        },
        {
          Effect   = "Allow"
          Action   = "ec2:DescribeInstances"
          Resource = "*"
        },
        {
          Effect = "Allow"
          Action = [
            "logs:CreateLogGroup",
            "logs:CreateLogStream",
            "logs:PutLogEvents"
          ]
          Resource = "*"
        },
        {
          "Effect" : "Allow",
          "Action" : [
            "route53:ListHostedZones"
          ],
          "Resource" : "*"
        }
      ],
      var.zone_id != "" ? [
        {
          Effect = "Allow"
          Action = [
            "route53:ListResourceRecordSets",
            "route53:ChangeResourceRecordSets"
          ]
          Resource = "arn:aws:route53:::hostedzone/${var.zone_id}"
        }
      ] : []
    )
  })
}

resource "aws_iam_role_policy" "lambda_s3_maps" {
  name = "${var.name}-lambda-s3-maps"
  role = aws_iam_role.lambda_exec.id

  policy = jsonencode({
    Version = "2012-10-17",
    Statement = [
      {
        Effect = "Allow",
        Action = [
          "s3:ListBucket",
          "s3:GetObject",
          "s3:GetObjectAttributes"
        ],
        Resource = [
          "arn:aws:s3:::${var.map_bucket_name}",
          "arn:aws:s3:::${var.map_bucket_name}/*"
        ]
      }
    ]
  })
}

data "archive_file" "lambda_zip" {
  type        = "zip"
  source_file = "${path.module}/lambda/handler.py"
  output_path = "${path.module}/lambda/lambda.zip"
}

resource "aws_lambda_function" "control" {
  function_name = "${var.name}-control"
  role          = aws_iam_role.lambda_exec.arn
  handler       = "handler.lambda_handler"
  runtime       = "python3.11"

  #   filename         = "${path.module}/lambda/lambda.zip"
  #   source_code_hash = filebase64sha256("${path.module}/lambda/lambda.zip")
  filename         = data.archive_file.lambda_zip.output_path
  source_code_hash = data.archive_file.lambda_zip.output_base64sha256

  environment {
    variables = {
      INSTANCE_ID = var.instance_id
      DNS_NAME    = var.dns_name
      CORS_ORIGIN = var.cors_origin
      ZONE_ID     = var.zone_id
    }
  }
}

resource "aws_lambda_function" "details" {
  function_name = "${var.name}-details"
  filename      = var.details_zip_path

  source_code_hash = filebase64sha256(var.details_zip_path)
  handler          = "app.main.lambda_handler"
  runtime          = "python3.11"
  role             = aws_iam_role.lambda_exec.arn

  environment {
    variables = {
    }
  }
}

resource "aws_lambda_function" "worlds" {
  function_name = "${var.name}-worlds"
  role          = aws_iam_role.lambda_exec.arn
  filename      = var.worlds_zip_path

  source_code_hash = filebase64sha256(var.worlds_zip_path)
  handler          = "app.main.lambda_handler"
  runtime          = "python3.11"
  timeout          = 10

  environment {
    variables = {
      MAPS_BUCKET = var.map_bucket_name
      CORS_ORIGIN = var.cors_origin
      MAP_PREFIX  = "maps/"
      BASE_URL    = ""
    }
  }
}

resource "aws_apigatewayv2_api" "http" {
  name          = "${var.name}-api"
  protocol_type = "HTTP"
}

resource "aws_apigatewayv2_integration" "lambda" {
  api_id             = aws_apigatewayv2_api.http.id
  integration_type   = "AWS_PROXY"
  integration_uri    = aws_lambda_function.control.invoke_arn
  integration_method = "POST"
}

resource "aws_apigatewayv2_integration" "worlds" {
  api_id                 = aws_apigatewayv2_api.http.id
  integration_type       = "AWS_PROXY"
  integration_uri        = aws_lambda_function.worlds.invoke_arn
  payload_format_version = "2.0"
}

resource "aws_apigatewayv2_stage" "default" {
  api_id      = aws_apigatewayv2_api.http.id
  name        = "$default"
  auto_deploy = true
}

resource "aws_apigatewayv2_route" "root" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "POST /{action}" # e.g. POST /start, POST /stop, POST /status
  target    = "integrations/${aws_apigatewayv2_integration.lambda.id}"
}

resource "aws_apigatewayv2_route" "status" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "ANY /status"
  target    = "integrations/${aws_apigatewayv2_integration.lambda.id}"
}

resource "aws_apigatewayv2_route" "start" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "ANY /start"
  target    = "integrations/${aws_apigatewayv2_integration.lambda.id}"
}

resource "aws_apigatewayv2_route" "stop" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "ANY /stop"
  target    = "integrations/${aws_apigatewayv2_integration.lambda.id}"
}

resource "aws_apigatewayv2_route" "proxy_root" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "ANY /{proxy+}"
  target    = "integrations/${aws_apigatewayv2_integration.lambda.id}"
}

resource "aws_apigatewayv2_route" "proxy_api" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "ANY /api/{proxy+}"
  target    = "integrations/${aws_apigatewayv2_integration.lambda.id}"
}

resource "aws_apigatewayv2_integration" "details" {
  api_id                 = aws_apigatewayv2_api.http.id
  integration_type       = "AWS_PROXY"
  integration_uri        = aws_lambda_function.details.invoke_arn
  integration_method     = "POST"
  payload_format_version = "2.0"
}

resource "aws_apigatewayv2_route" "details" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "ANY /api/details"
  target    = "integrations/${aws_apigatewayv2_integration.details.id}"
}

resource "aws_apigatewayv2_route" "worlds_list" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "GET /api/worlds"
  target    = "integrations/${aws_apigatewayv2_integration.worlds.id}"
}

resource "aws_apigatewayv2_route" "worlds_detail" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "GET /api/worlds/{name}"
  target    = "integrations/${aws_apigatewayv2_integration.worlds.id}"
}

resource "aws_apigatewayv2_route" "worlds_dimension" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "GET /api/worlds/{name}/{dimension}"
  target    = "integrations/${aws_apigatewayv2_integration.worlds.id}"
}

resource "aws_apigatewayv2_route" "worlds_api_cors" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "OPTIONS /api/worlds/{proxy+}"
  target    = "integrations/${aws_apigatewayv2_integration.worlds.id}"
}

resource "aws_apigatewayv2_route" "worlds_maps" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "GET /api/worlds/{world}/maps"
  target    = "integrations/${aws_apigatewayv2_integration.worlds.id}"
}

resource "aws_lambda_permission" "apigw_invoke_details" {
  statement_id  = "AllowAPIGatewayInvokeDetails"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.details.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.http.execution_arn}/*"
}


resource "aws_lambda_permission" "allow_api_worlds" {
  statement_id  = "AllowAPIGatewayInvokeWorlds"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.worlds.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.http.execution_arn}/*/*"
}

resource "aws_lambda_permission" "apigw" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.control.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.http.execution_arn}/*/*"
}
