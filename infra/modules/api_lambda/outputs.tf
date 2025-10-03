output "api_endpoint" {
  description = "Invoke URL for the Lambda API"
  value       = aws_apigatewayv2_api.http.api_endpoint
}
