variable "provider_token" {
    type        = string
    description = "API token for provider authentication"
    sensitive   = true
}

variable "ssh_key" {
    type        = string
    description = "ssh key"
    sensitive   = true
}

variable "root_password" {
    type        = string
    description = "Root access"
    sensitive   = true
}

