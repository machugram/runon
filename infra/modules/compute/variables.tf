variable "image_id" {
  type        = string
  description = "Local base image id. The image is built outside Terraform."
}

variable "network_name" {
  type        = string
  description = "Docker network name shared with the data tier."
}

variable "ssh_public_key_path" {
  type        = string
  description = "Host path of the lab public key mounted into each node."
}

variable "nodes" {
  type = list(object({
    name                = string
    role                = string
    ssh_host_port       = number
    publish_http        = bool
    http_host_port      = number
    http_container_port = number
  }))
  description = "SSH nodes Ansible will configure. publish_http is for the proxy only."

  validation {
    condition     = alltrue([for node in var.nodes : contains(["app", "proxy"], node.role)])
    error_message = "node role must be app or proxy."
  }
}
