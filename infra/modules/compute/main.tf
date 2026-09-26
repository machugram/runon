resource "docker_container" "node" {
  for_each = { for node in var.nodes : node.name => node }

  name       = each.value.name
  image      = var.image_id
  hostname   = each.value.name
  must_run   = true
  restart    = "unless-stopped"
  log_driver = "json-file"

  # Pin the daemon's log defaults so a later plan does not replace the container.
  log_opts = {
    "max-file" = "5"
    "max-size" = "20m"
  }

  networks_advanced {
    name = var.network_name
  }

  volumes {
    host_path      = var.ssh_public_key_path
    container_path = "/bootstrap/authorized_keys"
    read_only      = true
  }

  ports {
    internal = 22
    external = each.value.ssh_host_port
    ip       = "127.0.0.1"
    protocol = "tcp"
  }

  dynamic "ports" {
    for_each = each.value.publish_http ? [each.value] : []
    content {
      internal = ports.value.http_container_port
      external = ports.value.http_host_port
      ip       = "127.0.0.1"
      protocol = "tcp"
    }
  }
}
