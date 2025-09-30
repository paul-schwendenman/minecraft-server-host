output "instance_id" {
  value = aws_instance.minecraft.id
}

output "public_ip" {
  value = aws_instance.minecraft.public_ip
}

output "security_group_id" {
  value = aws_security_group.minecraft.id
}

output "world_volume_id" {
  value = aws_ebs_volume.world.id
}
