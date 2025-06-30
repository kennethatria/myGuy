terraform {
  required_version = ">= 1.0"
  
  required_providers {
    linode = {
      source  = "linode/linode"
      version = "~> 2.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.4"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "~> 4.0"
    }
  }

  # Configure remote state (recommended)
  backend "s3" {
    # Configure this with your Akamai Object Storage bucket
    bucket   = "myguy-terraform-state"
    key      = "terraform.tfstate"
    region   = "us-east-1"
    endpoint = "https://us-east-1.linodeobjects.com"
    
    # Set skip_credentials_validation and skip_region_validation to true for Linode Object Storage
    skip_credentials_validation = true
    skip_region_validation     = true
    skip_metadata_api_check    = true
  }
}

provider "linode" {
  # Set via environment variable: LINODE_TOKEN
  # token = var.linode_token
}