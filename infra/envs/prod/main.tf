locals {
  secrets_dir = abspath("${path.root}/../../../.local/${var.env_name}")
  nodes = [
    {
      name                = "${var.env_name}-proxy"
      role                = "proxy"
      ssh_host_port       = 2211
      publish_http        = true
      http_host_port      = var.proxy_host_port
      http_container_port = 80
    },
    {
      name                = "${var.env_name}-app-1"
      role                = "app"
      ssh_host_port       = 2212
      publish_http        = false
      http_host_port      = 0
      http_container_port = 8080
    },
    {
      name                = "${var.env_name}-app-2"
      role                = "app"
      ssh_host_port       = 2213
      publish_http        = false
      http_host_port      = 0
      http_container_port = 8080
    },
  ]
}

resource "tls_private_key" "lab" {
  algorithm = "ED25519"
}

resource "local_sensitive_file" "private_key" {
  content              = tls_private_key.lab.private_key_openssh
  filename             = "${local.secrets_dir}/id_ed25519"
  file_permission      = "0600"
  directory_permission = "0700"
}

resource "local_file" "public_key" {
  content              = tls_private_key.lab.public_key_openssh
  filename             = "${local.secrets_dir}/id_ed25519.pub"
  file_permission      = "0644"
  directory_permission = "0700"
}

resource "random_password" "db" {
  length  = 32
  special = false
}

data "external" "base_image" {
  program = ["docker", "image", "inspect", var.base_image, "--format", "{\"id\":\"{{.Id}}\"}"]
}

module "network" {
  source      = "../../modules/network"
  name_prefix = var.env_name
}

module "data" {
  source       = "../../modules/data"
  name_prefix  = var.env_name
  network_name = module.network.name
  db_name      = var.db_name
  db_user      = var.db_user
  db_password  = random_password.db.result
}

module "compute" {
  source              = "../../modules/compute"
  image_id            = data.external.base_image.result.id
  network_name        = module.network.name
  ssh_public_key_path = local_file.public_key.filename
  nodes               = local.nodes
}
