output "vpc_id" {
  description = "ID of the myGuy VPC"
  value       = linode_vpc.main.id
}

output "subnet_id" {
  description = "ID of the VPC subnet"
  value       = linode_vpc_subnet.main.id
}

output "nodebalancer_id" {
  description = "ID of the node balancer"
  value       = linode_nodebalancer.main.id
}

output "nodebalancer_ipv4" {
  description = "Public IPv4 address of the node balancer"
  value       = linode_nodebalancer.main.ipv4
}

output "instance_id" {
  description = "ID of the myGuy instance"
  value       = linode_instance.my_guy_instance.id
}

output "instance_ip_address" {
  description = "Public IP address of the myGuy instance"
  value       = linode_instance.my_guy_instance.ipv4[0]
}

output "instance_vpc_ip" {
  description = "Private VPC IP address of the myGuy instance"
  value       = local.instance_vpc_ip
}

output "instance_private_ip" {
  description = "Private IP address used by the nodebalancer to route traffic to the instance"
  value       = linode_instance.my_guy_instance.private_ip_address
}

output "firewall_id" {
  description = "ID of the firewall attached to the instance"
  value       = linode_firewall.my_firewall.id
}

output "nodebalancer_config_id" {
  description = "ID of the nodebalancer config"
  value       = linode_nodebalancer_config.main.id
}

output "nodebalancer_node_id" {
  description = "ID of the nodebalancer node"
  value       = linode_nodebalancer_node.main.id
}

output "zipkin_instance_id" {
  description = "ID of the Zipkin instance"
  value       = linode_instance.zipkin_instance.id
}

output "zipkin_instance_ip" {
  description = "Public IP of the Zipkin instance"
  value       = linode_instance.zipkin_instance.ipv4[0]
}

output "zipkin_vpc_ip" {
  description = "VPC IP of the Zipkin instance (use this for ZIPKIN_URL in app services)"
  value       = local.zipkin_vpc_ip
}

output "zipkin_url" {
  description = "Zipkin spans endpoint reachable from within the VPC"
  value       = "http://${local.zipkin_vpc_ip}:9411/api/v2/spans"
}
