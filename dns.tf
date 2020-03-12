resource "aws_route53_zone" "primary" {
  name = var.dns_name
}

resource "aws_route53_record" "minecraft" {
  zone_id = aws_route53_zone.primary.zone_id
  name    = var.dns_name
  type    = "A"
  ttl     = "300"
  records = ["${aws_instance.minecraft_server.public_ip}"]
}
