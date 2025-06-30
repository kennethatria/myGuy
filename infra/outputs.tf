# Instance Information
output "app_instance_ip" {
  description = "Public IP address of the application server"
  value       = linode_instance.app.ip_address
}

output "app_instance_private_ip" {
  description = "Private IP address of the application server"
  value       = linode_instance.app.private_ip_address
}

output "app_instance_id" {
  description = "ID of the application server"
  value       = linode_instance.app.id
}

# Database Information
output "database_host" {
  description = "Database host"
  value       = var.environment == "production" || !var.use_shared_database ? linode_database_postgresql.main[0].host : "shared-with-production"
  sensitive   = true
}

output "database_port" {
  description = "Database port"
  value       = var.environment == "production" || !var.use_shared_database ? linode_database_postgresql.main[0].port : 5432
}

output "database_username" {
  description = "Database username"
  value       = var.environment == "production" || !var.use_shared_database ? linode_database_postgresql.main[0].root_username : "shared"
  sensitive   = true
}

output "database_password" {
  description = "Database password"
  value       = var.postgres_password != "" ? var.postgres_password : random_password.postgres_password.result
  sensitive   = true
}

# Load Balancer Information (Production only)
output "load_balancer_ip" {
  description = "Load balancer IP address"
  value       = var.environment == "production" ? linode_nodebalancer.main[0].ipv4 : null
}

output "load_balancer_hostname" {
  description = "Load balancer hostname"
  value       = var.environment == "production" ? linode_nodebalancer.main[0].hostname : null
}

# Object Storage Information
output "bucket_name" {
  description = "Object storage bucket name"
  value       = linode_object_storage_bucket.uploads.label
}

output "bucket_region" {
  description = "Object storage bucket region"
  value       = linode_object_storage_bucket.uploads.region
}

output "bucket_hostname" {
  description = "Object storage bucket hostname"
  value       = "${linode_object_storage_bucket.uploads.label}.${var.linode_region}.linodeobjects.com"
}

# Application URLs
output "application_url" {
  description = "Application URL"
  value       = var.environment == "production" ? "https://${var.domain_name}" : "https://staging.${var.domain_name}"
}

output "staging_url" {
  description = "Direct staging server URL (bypass load balancer)"
  value       = var.environment == "staging" ? "http://${linode_instance.app.ip_address}" : null
}

# SSH Information
output "ssh_command" {
  description = "SSH command to connect to the application server"
  value       = "ssh root@${linode_instance.app.ip_address}"
}

# Environment Configuration
output "environment" {
  description = "Current environment"
  value       = var.environment
}

output "jwt_secret" {
  description = "JWT secret for application"
  value       = var.jwt_secret != "" ? var.jwt_secret : random_password.jwt_secret.result
  sensitive   = true
}

# VPC Information
output "vpc_id" {
  description = "VPC ID"
  value       = linode_vpc.main.id
}

output "subnet_id" {
  description = "Subnet ID"
  value       = linode_vpc_subnet.main.id
}

# Firewall Information
output "firewall_id" {
  description = "Firewall ID"
  value       = linode_firewall.web.id
}

# Cost Estimation (approximate)
output "estimated_monthly_cost_eur" {
  description = "Estimated monthly cost in EUR"
  value = var.environment == "production" ? {
    compute      = "9.00"
    database     = "15.00"
    storage      = "5.00"
    loadbalancer = "10.00"
    total        = "39.00"
  } : {
    compute      = "4.50"
    database     = "0.00"  # Shared with production
    storage      = "2.50"
    loadbalancer = "0.00"  # No load balancer for staging
    total        = "7.00"
  }
}

# Deployment Information
output "deployment_info" {
  description = "Key deployment information"
  value = {
    environment           = var.environment
    region               = var.linode_region
    instance_type        = var.environment == "production" ? var.production_instance_type : var.app_instance_type
    uses_shared_database = var.use_shared_database
    has_load_balancer    = var.environment == "production"
  }
}