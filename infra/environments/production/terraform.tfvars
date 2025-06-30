# Production Environment Configuration
environment = "production"
project_name = "myguy"
domain_name = "myguy.work"

# Linode Configuration
linode_region = "eu-west"  # London region for EU
app_instance_type = "g6-standard-1"     # 2GB RAM, 1 CPU, €9/month
production_instance_type = "g6-standard-1"

# Production Settings
use_shared_database = false    # Dedicated database for production
enable_backups = false        # Keep disabled for MVP to save costs
enable_monitoring = false     # Basic monitoring only

# Database Configuration
database_engine_version = "15"
database_instance_type = "g6-nanode-1"  # 1GB database instance

# Security note: Set these via environment variables or separate .tfvars.secret file
# linode_token = "your_linode_api_token"
# ssh_public_key = "ssh-rsa AAAAB3NzaC1yc2E..."
# jwt_secret = "your_production_jwt_secret"
# postgres_password = "your_secure_production_password"
# github_token = "your_github_token"