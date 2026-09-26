resource "docker_network" "this" {
  name = "${var.name_prefix}-platform"
}
