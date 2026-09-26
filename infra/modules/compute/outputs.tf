output "nodes" {
  value = [for node in var.nodes : {
    name          = node.name
    role          = node.role
    ssh_host_port = node.ssh_host_port
  }]
  description = "Nodes Ansible should inventory. ssh_host_port is published on 127.0.0.1."
}
