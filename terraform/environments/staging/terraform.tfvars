# Staging Environment Configuration
environment = "staging"
project_name = "myguy"
domain_name = "myguy.work"

# Linode Configuration
linode_region = "eu-west"  # London region for EU
app_instance_type = "g6-nanode-1"  # 1GB RAM, €4.50/month

# Cost Optimization Settings
use_shared_database = true      # Share production database
enable_backups = false         # Disable backups for cost savings
enable_monitoring = false      # Basic monitoring only

# Database Configuration
database_engine_version = "15"
database_instance_type = "g6-nanode-1"

# You'll need to set these via environment variables or terraform.tfvars.secret
# linode_token = "your_linode_api_token"
# ssh_public_key = "ssh-rsa AAAAB3NzaC1yc2E..."
# jwt_secret = "your_jwt_secret_here"
# postgres_password = "your_secure_password"
# github_token = "your_github_token"