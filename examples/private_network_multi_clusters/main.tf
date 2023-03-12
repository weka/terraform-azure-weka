provider "azurerm" {
  subscription_id = var.subscription_id
  partner_id      = var.weka_partner_id
  features {
  }
}

module "create-network" {
  source            = "../../modules/create_networks"
  prefix            = var.prefix
  rg_name           = var.rg_name
  address_space     = var.address_space
  subnet_prefixes   = var.subnet_prefixes
  subnet_delegation = var.subnet_delegation
  private_network   = var.private_network
}

module "deploy-weka" {
  count                 = length(var.clusters_list)
  source                = "../.."
  prefix                = var.prefix
  rg_name               = var.rg_name
  vnet_name             = module.create-network.vnet-name
  subnets               = module.create-network.subnets-name
  sg_id                 = module.create-network.sg-id
  subnet_delegation_id  = module.create-network.subnet-delegation-id
  apt_repo_url          = var.apt_repo_url
  private_network       = var.private_network
  install_weka_url      = var.install_weka_url
  cluster_name          = var.clusters_list[count.index]
  install_ofed_url      = var.install_ofed_url
  instance_type         = var.instance_type
  cluster_size          = var.cluster_size
  set_obs_integration   = var.set_obs_integration
  tiering_ssd_percent   = var.tiering_ssd_percent
  subscription_id       = var.subscription_id
  private_dns_zone_name = module.create-network.private-dns-zone-name
  depends_on            = [module.create-network]
}