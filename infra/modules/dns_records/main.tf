resource "aws_route53_record" "a_record" {
  count   = var.create_a_record ? 1 : 0
  zone_id = var.zone_id
  name    = var.dns_name
  type    = "A"
  ttl     = var.ttl
  records = var.ipv4_addresses

  lifecycle {
    ignore_changes = [
      records
    ]
  }
}

# --- AAAA record (IPv6) ---
resource "aws_route53_record" "aaaa_record" {
  count   = var.create_aaaa_record ? 1 : 0
  zone_id = var.zone_id
  name    = var.dns_name
  type    = "AAAA"
  ttl     = var.ttl
  records = var.ipv6_addresses
}
