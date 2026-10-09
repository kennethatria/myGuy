variable "provider_token" {
  type        = string
  description = "API token for provider authentication"
  sensitive   = true
}

variable "authorized_keys" {
  type        = string
  description = "Single-line SSH public key (contents of the .pub file) installed for root"
  sensitive   = true

  validation {
    condition     = can(regex("^(ssh-ed25519|ssh-rsa|ecdsa-sha2-nistp[0-9]+|sk-ecdsa-sha2-nistp256@openssh.com) [A-Za-z0-9+/=]+( [^\\n]*)?$", trimspace(var.authorized_keys)))
    error_message = "authorized_keys must be a one-line SSH public key (the .pub file), not a private key."
  }
}

variable "root_password" {
  type        = string
  description = "Root access"
  sensitive   = true
}

variable "region" {
  type        = string
  description = "Region with resources"
  default     = "de-fra-2"
}

variable "infra_name" {
  type        = string
  description = "default infra name"
  default     = "myguy"
}

variable "environment" {
  type        = string
  description = "default infra name"
  default     = "dev"
}

