output "name" {
  value       = docker_network.this.name
  description = "Docker network name. Containers on this network resolve each other by container name."
}
