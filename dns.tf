resource "aws_route53_zone" "primary" {
  name = var.dns_name
}

resource "aws_route53_record" "minecraft" {
  zone_id = aws_route53_zone.primary.zone_id
  name    = var.dns_name
  type    = "A"
  ttl     = "300"
  records = [aws_instance.minecraft_server.public_ip]
}

resource "aws_route53_record" "www" {
  zone_id = aws_route53_zone.primary.zone_id
  name    = var.webapp_dns_name
  type    = "A"

  alias {
    name                   = aws_cloudfront_distribution.www_distribution.domain_name
    zone_id                = aws_cloudfront_distribution.www_distribution.hosted_zone_id
    evaluate_target_health = false
  }
}
