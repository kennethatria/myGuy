locals {
  instance_vpc_ip = "10.0.0.2"
  zipkin_vpc_ip   = "10.0.0.3"
}

resource "linode_vpc" "main" {
  label       = "${var.infra_name}-${var.environment}"
  region      = var.region
  description = "myGuy VPC"
}

resource "linode_vpc_subnet" "main" {
  vpc_id = linode_vpc.main.id
  label  = "${var.infra_name}-${var.environment}-subnet"
  ipv4   = "10.0.0.0/24"
}

resource "linode_nodebalancer" "main" {
  label  = "${var.infra_name}-${var.environment}-nodebalancer"
  region = var.region
  client_conn_throttle = 20
  client_udp_sess_throttle = 10
  tags = ["dev"]
}

resource "linode_nodebalancer_config" "main" {
  nodebalancer_id = linode_nodebalancer.main.id
  port            = 80
  protocol        = "http"
  check           = "http_body"
  check_path      = "/healthcheck/"
  check_body      = "healthcheck"
  check_attempts  = 1
  check_timeout   = 25
  check_interval  = 30
  stickiness      = "http_cookie"
  algorithm       = "roundrobin"
}

resource "linode_nodebalancer_node" "main" {
  nodebalancer_id = linode_nodebalancer.main.id
  config_id       = linode_nodebalancer_config.main.id
  label           = "${var.infra_name}-${var.environment}-node"
  address         = "${linode_instance.my_guy_instance.private_ip_address}:80"
  mode            = "accept"
  weight          = 100

  lifecycle {
    replace_triggered_by = [linode_instance.my_guy_instance]
  }
}

resource "linode_firewall" "my_firewall" {
  label = "${var.infra_name}-${var.environment}-firewall"

  inbound {
    label    = "allow-http"
    action   = "ACCEPT"
    protocol = "TCP"
    ports    = "80"
    ipv4     = ["0.0.0.0/0"]
    ipv6     = ["::/0"]
  }

  inbound {
    label    = "allow-ssh"
    action   = "ACCEPT"
    protocol = "TCP"
    ports    = "22"
    ipv4     = ["0.0.0.0/0"]
    ipv6     = ["::/0"]
  }

  inbound {
    label    = "allow-node-exporter-from-vpc"
    action   = "ACCEPT"
    protocol = "TCP"
    ports    = "9100"
    ipv4     = ["10.0.0.0/24"]
  }

  inbound {
    label    = "allow-falco-metrics-from-vpc"
    action   = "ACCEPT"
    protocol = "TCP"
    ports    = "8765"
    ipv4     = ["10.0.0.0/24"]
  }

  inbound_policy  = "DROP"
  outbound_policy = "ACCEPT"

  linodes = [linode_instance.my_guy_instance.id]
}


resource "linode_firewall" "zipkin_firewall" {
  label = "${var.infra_name}-${var.environment}-monitoring-firewall"

  inbound {
    label    = "allow-zipkin-from-vpc"
    action   = "ACCEPT"
    protocol = "TCP"
    ports    = "9411"
    ipv4     = ["10.0.0.0/24"]
  }

  inbound {
    label    = "allow-prometheus-from-vpc"
    action   = "ACCEPT"
    protocol = "TCP"
    ports    = "9090"
    ipv4     = ["10.0.0.0/24"]
  }

  inbound {
    label    = "allow-grafana-from-vpc"
    action   = "ACCEPT"
    protocol = "TCP"
    ports    = "3000"
    ipv4     = ["10.0.0.0/24"]
  }

  inbound {
    label    = "allow-ssh-from-vpc"
    action   = "ACCEPT"
    protocol = "TCP"
    ports    = "22"
    ipv4     = ["10.0.0.0/24"]
  }

  inbound_policy  = "DROP"
  outbound_policy = "ACCEPT"

  linodes = [linode_instance.zipkin_instance.id]
}

resource "linode_instance" "zipkin_instance" {
  image           = "linode/ubuntu24.04"
  label           = "${var.infra_name}-${var.environment}-zipkin"
  tags            = [var.environment, "zipkin"]
  region          = var.region
  type            = "g6-nanode-1"
  authorized_keys = [var.authorized_keys]
  root_pass       = var.root_password
  private_ip      = true

  metadata {
    user_data = base64encode(<<-EOF
      #cloud-config
      write_files:
        - path: /etc/netplan/05-vpc.yaml
          content: |
            network:
              version: 2
              renderer: networkd
              ethernets:
                eth1:
                  dhcp4: false
                  addresses:
                    - ${local.zipkin_vpc_ip}/24
      runcmd:
        - netplan apply
    EOF
    )
  }

  interface {
    purpose = "public"
  }

  interface {
    purpose   = "vpc"
    subnet_id = linode_vpc_subnet.main.id
    ipv4 {
      vpc = local.zipkin_vpc_ip
    }
  }
}

resource "linode_instance" "my_guy_instance" {
  image           = "linode/ubuntu24.04"
  label           = "${var.infra_name}-${var.environment}-instance"
  tags            = [var.environment]
  region          = var.region
  type            = "g6-nanode-1"
  authorized_keys = [var.authorized_keys]
  root_pass       = var.root_password
  private_ip      = true

  interface {
    purpose = "public"
  }

  interface {
    purpose      = "vpc"
    subnet_id    = linode_vpc_subnet.main.id
    ipam_address = "${local.instance_vpc_ip}/24"
  }
}
