provider "azurerm" {
  subscription_id = var.subscription_id
  features {
  }
}

module "create-network" {
  source            = "../../modules/create_networks"
  prefix            = var.prefix
  rg_name           = var.rg_name
  address_space     = var.address_space
  subnet_delegation = var.subnet_delegation
  subnet_prefixes   = var.subnet_prefixes
}

module "deploy-weka" {
  source                 = "../.."
  prefix                 = var.prefix
  rg_name                = var.rg_name
  vnet_name              = module.create-network.vnet-name
  subnets                = module.create-network.subnets-name
  sg_id                  = module.create-network.sg-id
  get_weka_io_token      = var.get_weka_io_token
  cluster_name           = var.cluster_name
  subnet_delegation_id   = module.create-network.subnet-delegation-id
  set_obs_integration    = var.set_obs_integration
  instance_type          = var.instance_type
  cluster_size           = var.cluster_size
  tiering_ssd_percent    = var.tiering_ssd_percent
  subscription_id        = var.subscription_id
  single_placement_group = var.single_placement_group
  private_dns_zone_name  = module.create-network.private-dns-zone-name
  depends_on             = [module.create-network]
}
