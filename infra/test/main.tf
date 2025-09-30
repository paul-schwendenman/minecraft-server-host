data "aws_vpc" "default" {
  default = true
}

data "aws_subnets" "default" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.default.id]
  }
}

module "mc_stack" {
  source           = "../modules/mc_stack"
  name             = "minecraft-test"
  ami_id           = "ami-033bf9694c0d2ea06"
  instance_type    = "t3.small"
  vpc_id           = data.aws_vpc.default.id
  subnet_id        = tolist(data.aws_subnets.default.ids)[0]
  key_name         = "minecraft-packer"
  root_volume_size = 8
  ssh_cidr_blocks  = ["104.230.245.46/32"]
  world_version    = "1.21.8"
}
