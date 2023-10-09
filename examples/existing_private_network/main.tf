provider "azurerm" {
  subscription_id = var.subscription_id
  partner_id      = "f13589d1-f10d-4c3b-ae42-3b1a8337eaf1"
  features {
  }
}

module "weka_deployment" {
  source                = "../.."
  prefix                = "weka"
  rg_name               = "weka-rg"
  cluster_name          = "poc"
  subnet_name           = "weka-subnet"
  vnet_name             = "weka-vnet"
  vnet_rg_name          = "weka-rg"
  sg_id                 = "/subscriptions/../resourceGroups/../providers/Microsoft.Network/networkSecurityGroups/.."
  apt_repo_server       = "http://11.0.0.4/ubuntu/mirror/archive.ubuntu.com/ubuntu/"
  install_weka_url      = "..."
  subnet_delegation_id  = "/subscriptions/../resourceGroups/../providers/Microsoft.Network/virtualNetworks/../subnets/.."
  private_network       = true
  assign_public_ip      = true
  cluster_size          = 6
  set_obs_integration   = true
  tiering_ssd_percent   = 20
  subscription_id       = var.subscription_id
  private_dns_zone_name = "weka.private.net"
  private_dns_rg_name   = "dns-weka-rg"
}
