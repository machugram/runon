variable "name_prefix" {
  type        = string
  description = "Environment name used in the container and volume names."
}

variable "network_name" {
  type        = string
  description = "Docker network name shared with the compute tier."
}

variable "db_name" {
  type        = string
  description = "Database created by the Postgres image on first start."
}

variable "db_user" {
  type        = string
  description = "Admin role created by the Postgres image on first start."
}

variable "db_password" {
  type        = string
  sensitive   = true
  description = "Admin password. Stored in Terraform state, not in git."
}

variable "postgres_image" {
  type        = string
  description = "Postgres image to pull."
  default     = "postgres:16"
}
