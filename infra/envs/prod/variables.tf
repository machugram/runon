variable "env_name" {
  type        = string
  description = "Environment name. Selects container names, ports, and the secrets directory."

  validation {
    condition     = contains(["dev", "prod"], var.env_name)
    error_message = "env_name must be dev or prod."
  }
}

variable "base_image" {
  type        = string
  description = "Local image tag built by the platform CLI before apply."
  default     = "platform-lab-base:local"
}

variable "proxy_host_port" {
  type        = number
  description = "Localhost port published by the proxy."

  validation {
    condition     = var.proxy_host_port > 1024 && var.proxy_host_port < 65536
    error_message = "proxy_host_port must be an unprivileged port."
  }
}

variable "db_user" {
  type        = string
  description = "Postgres admin role and application role."
  default     = "platform"
}

variable "db_name" {
  type        = string
  description = "Postgres database name."
  default     = "platform"
}
