resource "docker_image" "postgres" {
  name         = var.postgres_image
  keep_locally = true
}

resource "docker_volume" "data" {
  name = "${var.name_prefix}-postgres"
}

resource "docker_container" "postgres" {
  name       = "${var.name_prefix}-postgres"
  image      = docker_image.postgres.image_id
  must_run   = true
  restart    = "unless-stopped"
  log_driver = "json-file"

  # Pin the daemon's log defaults so a later plan does not replace the container.
  log_opts = {
    "max-file" = "5"
    "max-size" = "20m"
  }

  env = [
    "POSTGRES_USER=${var.db_user}",
    "POSTGRES_PASSWORD=${var.db_password}",
    "POSTGRES_DB=${var.db_name}",
  ]

  networks_advanced {
    name = var.network_name
  }

  volumes {
    volume_name    = docker_volume.data.name
    container_path = "/var/lib/postgresql/data"
    read_only      = false
  }
}
