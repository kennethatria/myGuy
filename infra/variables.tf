# Core Configuration
variable "project_name" {
  description = "Name of the project"
  type        = string
  default     = "myguy"
}

variable "domain_name" {
  description = "Domain name for the application"
  type        = string
  default     = "myguy.work"
}

variable "environment" {
  description = "Environment (staging or production)"
  type        = string
  validation {
    condition     = contains(["staging", "production"], var.environment)
    error_message = "Environment must be either 'staging' or 'production'."
  }
}

variable "ssl_cert" {
  description = "SSL Certificate content"
  type        = string
  sensitive   = true
  default     = ""
}

variable "ssl_key" {
  description = "SSL Private Key content"
  type        = string
  sensitive   = true
  default     = ""
}

# Linode Configuration
variable "linode_region" {
  description = "Linode region for deployment"
  type        = string
  default     = "eu-west"  # London - closest to Europe for better latency
}

variable "linode_token" {
  description = "Linode API token"
  type        = string
  sensitive   = true
}

# Instance Configuration
variable "app_instance_type" {
  description = "Linode instance type for application servers"
  type        = string
  default     = "g6-nanode-1"  # 1GB RAM, 1 CPU, €4.50/month
}

variable "production_instance_type" {
  description = "Larger instance type for production"
  type        = string
  default     = "g6-standard-1"  # 2GB RAM, 1 CPU, €9/month
}

# Database Configuration
variable "database_engine_version" {
  description = "PostgreSQL version"
  type        = string
  default     = "15"
}

variable "database_instance_type" {
  description = "Database instance type"
  type        = string
  default     = "g6-nanode-1"  # 1GB for cost optimization
}

# SSH Key Configuration
variable "ssh_public_key" {
  description = "SSH public key for instance access"
  type        = string
}

# Application Configuration
variable "jwt_secret" {
  description = "JWT secret for authentication"
  type        = string
  sensitive   = true
}

variable "postgres_password" {
  description = "PostgreSQL password"
  type        = string
  sensitive   = true
}

# GitHub Configuration
variable "github_token" {
  description = "GitHub token for deployment actions"
  type        = string
  sensitive   = true
}

# Cost optimization flags
variable "use_shared_database" {
  description = "Use shared database for staging environment"
  type        = bool
  default     = true
}

variable "enable_backups" {
  description = "Enable automated backups"
  type        = bool
  default     = false  # Disable for cost savings in MVP
}

variable "enable_monitoring" {
  description = "Enable monitoring services"
  type        = bool
  default     = false  # Basic monitoring only
}