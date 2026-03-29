variable "provider_token" {
    type        = string
    description = "API token for provider authentication"
    sensitive   = true
}

variable "authorized_keys" {
    type        = string
    description = "ssh key"
    sensitive   = true
}

variable "root_password" {
    type        = string
    description = "Root access"
    sensitive   = true
}

