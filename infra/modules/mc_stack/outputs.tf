output "instance_id" {
  value = aws_instance.minecraft.id
}

output "public_ip" {
  value = aws_instance.minecraft.public_ip
}

output "private_ip" {
  description = "Private IPv4 address of the Minecraft server"
  value       = aws_instance.minecraft.private_ip
}

output "ipv6_addresses" {
  description = "IPv6 addresses assigned to the Minecraft server"
  value       = aws_instance.minecraft.ipv6_addresses
}

output "security_group_id" {
  value = aws_security_group.minecraft.id
}

output "world_volume_id" {
  value = aws_ebs_volume.world.id
}
