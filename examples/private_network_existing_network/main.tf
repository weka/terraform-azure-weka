provider "azurerm" {
  subscription_id = var.subscription_id
  features {
  }
}

module "create-network" {
  source            = "../../modules/create_networks"
  prefix            = var.prefix
  rg_name           = var.rg_name
  vnet_name         = var.vnet_name
  subnets_name_list = var.subnets_name_list
}

module "deploy-weka" {
  source                = "../.."
  prefix                = var.prefix
  rg_name               = var.rg_name
  vnet_name             = module.create-network.vnet-name
  subnets               = module.create-network.subnets-name
  sg_id                 = module.create-network.sg-id
  cluster_name          = var.cluster_name
  apt_repo_url          = var.apt_repo_url
  private_network       = var.private_network
  install_weka_url      = var.install_weka_url
  install_ofed_url      = var.install_ofed_url
  instance_type         = var.instance_type
  cluster_size          = var.cluster_size
  set_obs_integration   = var.set_obs_integration
  tiering_ssd_percent   = var.tiering_ssd_percent
  subscription_id       = var.subscription_id
  private_dns_zone_name = module.create-network.private-dns-zone-name
  depends_on            = [module.create-network]
}