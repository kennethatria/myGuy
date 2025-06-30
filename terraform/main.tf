# Generate random password for PostgreSQL if not provided
resource "random_password" "postgres_password" {
  length  = 16
  special = true
}

# Generate random JWT secret if not provided
resource "random_password" "jwt_secret" {
  length  = 32
  special = false
}

# SSH Key for instance access
resource "linode_sshkey" "main" {
  label   = "${var.project_name}-${var.environment}-key"
  ssh_key = var.ssh_public_key
}

# Create a VPC for network isolation
resource "linode_vpc" "main" {
  label       = "${var.project_name}-${var.environment}-vpc"
  region      = var.linode_region
  description = "VPC for ${var.project_name} ${var.environment} environment"
}

# VPC Subnet
resource "linode_vpc_subnet" "main" {
  vpc_id = linode_vpc.main.id
  label  = "${var.project_name}-${var.environment}-subnet"
  ipv4   = "10.0.0.0/24"
}

# Security configuration via Linode Firewall
resource "linode_firewall" "web" {
  label = "${var.project_name}-${var.environment}-web-firewall"

  # Allow SSH
  inbound {
    label    = "allow-ssh"
    action   = "ACCEPT"
    protocol = "TCP"
    ports    = "22"
    ipv4     = ["0.0.0.0/0"]
    ipv6     = ["::/0"]
  }

  # Allow HTTP
  inbound {
    label    = "allow-http"
    action   = "ACCEPT"
    protocol = "TCP"
    ports    = "80"
    ipv4     = ["0.0.0.0/0"]
    ipv6     = ["::/0"]
  }

  # Allow HTTPS
  inbound {
    label    = "allow-https"
    action   = "ACCEPT"
    protocol = "TCP"
    ports    = "443"
    ipv4     = ["0.0.0.0/0"]
    ipv6     = ["::/0"]
  }

  # Allow application ports (for debugging - remove in production)
  inbound {
    label    = "allow-app-ports"
    action   = "ACCEPT"
    protocol = "TCP"
    ports    = "8080-8082"
    ipv4     = ["0.0.0.0/0"]
    ipv6     = ["::/0"]
  }

  # Allow frontend port (for debugging - remove in production)
  inbound {
    label    = "allow-frontend"
    action   = "ACCEPT"
    protocol = "TCP"
    ports    = "5173"
    ipv4     = ["0.0.0.0/0"]
    ipv6     = ["::/0"]
  }

  # Outbound rules (allow all outbound traffic)
  outbound {
    label    = "allow-all-outbound"
    action   = "ACCEPT"
    protocol = "TCP"
    ports    = "1-65535"
    ipv4     = ["0.0.0.0/0"]
    ipv6     = ["::/0"]
  }

  outbound {
    label    = "allow-all-outbound-udp"
    action   = "ACCEPT"
    protocol = "UDP"
    ports    = "1-65535"
    ipv4     = ["0.0.0.0/0"]
    ipv6     = ["::/0"]
  }
}

# PostgreSQL Database (only create for production or if not using shared database)
resource "linode_database_postgresql" "main" {
  count = var.environment == "production" || !var.use_shared_database ? 1 : 0

  label           = "${var.project_name}-${var.environment}-db"
  engine_id       = "postgresql/${var.database_engine_version}"
  region          = var.linode_region
  type            = var.database_instance_type
  cluster_size    = 1
  encrypted       = false  # Disable encryption for cost savings
  ssl_connection  = true
  
  allow_list = [
    linode_instance.app.ip_address
  ]
}

# Object Storage Bucket for file uploads
resource "linode_object_storage_bucket" "uploads" {
  label   = "${var.project_name}-${var.environment}-uploads"
  region  = var.linode_region
  
  lifecycle_rule {
    id      = "delete_old_uploads"
    enabled = true
    
    expiration {
      days = 365  # Delete files after 1 year to control storage costs
    }
  }
}

# Application Server Instance
resource "linode_instance" "app" {
  label           = "${var.project_name}-${var.environment}-app"
  image           = "linode/ubuntu22.04"
  region          = var.linode_region
  type            = var.environment == "production" ? var.production_instance_type : var.app_instance_type
  authorized_keys = [linode_sshkey.main.ssh_key]
  
  # Add to VPC
  interface {
    purpose     = "vpc"
    vpc_id      = linode_vpc.main.id
    subnet_id   = linode_vpc_subnet.main.id
    primary     = true
  }

  # User data script for initial setup
  user_data = base64encode(templatefile("${path.module}/scripts/setup.sh", {
    environment     = var.environment
    domain_name     = var.environment == "production" ? var.domain_name : "staging.${var.domain_name}"
    jwt_secret      = var.jwt_secret != "" ? var.jwt_secret : random_password.jwt_secret.result
    postgres_host   = var.environment == "production" || !var.use_shared_database ? linode_database_postgresql.main[0].host : data.terraform_remote_state.production[0].outputs.database_host
    postgres_password = var.postgres_password != "" ? var.postgres_password : random_password.postgres_password.result
    bucket_name     = linode_object_storage_bucket.uploads.label
    bucket_region   = var.linode_region
  }))

  tags = [
    var.environment,
    var.project_name,
    "app-server"
  ]
}

# Attach firewall to instance
resource "linode_firewall_device" "app" {
  firewall_id = linode_firewall.web.id
  entity_id   = linode_instance.app.id
  entity_type = "linode"
}

# Load Balancer (only for production to save costs)
resource "linode_nodebalancer" "main" {
  count  = var.environment == "production" ? 1 : 0
  
  label  = "${var.project_name}-${var.environment}-lb"
  region = var.linode_region
  
  tags = [
    var.environment,
    var.project_name,
    "load-balancer"
  ]
}

resource "linode_nodebalancer_config" "http" {
  count = var.environment == "production" ? 1 : 0
  
  nodebalancer_id = linode_nodebalancer.main[0].id
  port            = 80
  protocol        = "http"
  check           = "http"
  check_path      = "/health"
  check_attempts  = 3
  check_timeout   = 10
  check_interval  = 15
  stickiness      = "none"
  algorithm       = "roundrobin"
}

resource "linode_nodebalancer_config" "https" {
  count = var.environment == "production" ? 1 : 0
  
  nodebalancer_id = linode_nodebalancer.main[0].id
  port            = 443
  protocol        = "https"
  check           = "http"
  check_path      = "/health"
  check_attempts  = 3
  check_timeout   = 10
  check_interval  = 15
  stickiness      = "none"
  algorithm       = "roundrobin"
  
  ssl_cert = file("${path.module}/ssl/cert.pem")
  ssl_key  = file("${path.module}/ssl/private.key")
}

resource "linode_nodebalancer_node" "app" {
  count = var.environment == "production" ? 1 : 0
  
  nodebalancer_id = linode_nodebalancer.main[0].id
  config_id       = linode_nodebalancer_config.http[0].id
  label           = "${var.project_name}-${var.environment}-app-node"
  address         = "${linode_instance.app.private_ip_address}:80"
  weight          = 100
  mode            = "accept"
}

# Data source to get production database info for staging
data "terraform_remote_state" "production" {
  count   = var.environment == "staging" && var.use_shared_database ? 1 : 0
  backend = "s3"
  
  config = {
    bucket   = "myguy-terraform-state"
    key      = "production/terraform.tfstate"
    region   = "us-east-1"
    endpoint = "https://us-east-1.linodeobjects.com"
    skip_credentials_validation = true
    skip_region_validation     = true
    skip_metadata_api_check    = true
  }
}