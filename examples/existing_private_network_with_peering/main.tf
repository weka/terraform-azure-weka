provider "azurerm" {
  subscription_id = var.subscription_id
  partner_id      = "f13589d1-f10d-4c3b-ae42-3b1a8337eaf1"
  features {
  }
}

module "create-network" {
  source            = "../../modules/create_networks"
  prefix            = var.prefix
  rg_name           = var.rg_name
  vnet_name         = var.vnet_name
  subnet_name       = var.subnet_name
  private_network   = var.private_network
}

module "peering" {
  source          = "../../modules/peering_vnets"
  prefix          = var.prefix
  rg_name         = var.rg_name
  vnet_name       = module.create-network.vnet-name
  vnet_to_peering = var.vnet_to_peering
  depends_on      = [module.create-network]
}

module "deploy-weka" {
  source                = "../.."
  prefix                = var.prefix
  rg_name               = var.rg_name
  vnet_name             = module.create-network.vnet-name
  vnet_rg_name          = module.create-network.vnet_rg_name
  subnet_name           = module.create-network.subnets-name
  sg_id                 = module.create-network.sg-id
  subnet_delegation     = var.subnet_delegation
  apt_repo_url          = var.apt_repo_url
  private_network       = var.private_network
  install_weka_url      = var.install_weka_url
  cluster_name          = var.cluster_name
  instance_type         = var.instance_type
  cluster_size          = var.cluster_size
  set_obs_integration   = var.set_obs_integration
  tiering_ssd_percent   = var.tiering_ssd_percent
  subscription_id       = var.subscription_id
  private_dns_zone_name = module.create-network.private-dns-zone-name
  depends_on            = [module.create-network, module.peering]
}