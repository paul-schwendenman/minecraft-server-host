terraform {
  cloud {
    organization = "whatsdoom"
    workspaces {
      name = "minecraft-server"
    }
  }
}
