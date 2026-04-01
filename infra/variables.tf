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

variable "region" {
    type = string
    description = "Region with resources"
    default = "de-fra-2"
}

variable "infra_name" {
    type = string
    description = "default infra name"
    default = "myguy"
}

variable "environment" {
    type = string
    description = "default infra name"
    default = "dev"
}

