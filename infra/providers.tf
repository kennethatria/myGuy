terraform {
  cloud {
    organization = "myGuy"

    workspaces {
      name = "dev-myguy"
    }
  }

  required_providers {
    linode = {
      source  = "linode/linode"
      version = "3.0.0"
    }
  }
}

provider "linode" {
  token = var.provider_token
}