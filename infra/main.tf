resource "linode_instance" "my_guy_instance" {
  image           = "linode/ubuntu24.04"
  label           = "myGuy-dev-instance"
  tags            = ["dev"]
  region          = "eu-central"
  type            = "g6-nanode-1"
  authorized_keys = [var.authorized_keys]
  root_pass       = var.root_password
}