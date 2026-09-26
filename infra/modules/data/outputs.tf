output "host" {
  value       = docker_container.postgres.name
  description = "DNS name of Postgres on the platform network."
}

output "port" {
  value       = 5432
  description = "Postgres port on the platform network. It is not published to the host."
}
