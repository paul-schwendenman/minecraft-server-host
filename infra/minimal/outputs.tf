output "instance_id" {
  value = aws_instance.minecraft.id
}

output "public_ip" {
  value = aws_instance.minecraft.public_ip
}

output "public_dns" {
  value = aws_instance.minecraft.public_dns
}
