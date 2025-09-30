module "mc_stack" {
  source          = "../../modules/mc_stack"
  name            = "minecraft-test"
  ami_id          = "ami-033bf9694c0d2ea06"
  instance_type   = "t3.small"
  vpc_id          = aws_vpc.main.id
  subnet_id       = aws_subnet.main.id
  key_name        = "minecraft-packer"
  root_volume_size = 8
  ssh_cidr_blocks = ["104.230.245.46/32"]
}
