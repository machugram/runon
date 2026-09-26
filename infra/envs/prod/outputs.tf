output "inventory" {
  description = "Ansible groups. Hostnames match Docker DNS names."
  value = {
    proxy = [for node in module.compute.nodes : {
      name = node.name
      host = "127.0.0.1"
      port = node.ssh_host_port
      user = "root"
    } if node.role == "proxy"]
    app = [for node in module.compute.nodes : {
      name = node.name
      host = "127.0.0.1"
      port = node.ssh_host_port
      user = "root"
    } if node.role == "app"]
  }
}

output "ansible" {
  description = "Values the CLI writes to the gitignored Ansible extra-vars file."
  sensitive   = true
  value = {
    env_name             = var.env_name
    db_host              = module.data.host
    db_port              = module.data.port
    db_admin_user        = var.db_user
    db_admin_password    = random_password.db.result
    db_name              = var.db_name
    db_user              = var.db_user
    db_password          = random_password.db.result
    ssh_private_key_path = local_sensitive_file.private_key.filename
    proxy_http_port      = var.proxy_host_port
  }
}
