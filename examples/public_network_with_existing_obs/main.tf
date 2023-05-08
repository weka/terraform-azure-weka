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
  address_space     = var.address_space
  subnet_delegation = var.subnet_delegation
  subnet_prefixes   = var.subnet_prefixes
}

module "deploy-weka" {
  source                = "../.."
  prefix                = var.prefix
  rg_name               = var.rg_name
  vnet_name             = module.create-network.vnet-name
  vnet_rg_name          = module.create-network.vnet_rg_name
  subnets               = module.create-network.subnets-name
  sg_id                 = module.create-network.sg-id
  subnet_delegation_id  = module.create-network.subnet-delegation-id
  get_weka_io_token     = var.get_weka_io_token
  cluster_name          = var.cluster_name
  set_obs_integration   = var.set_obs_integration
  instance_type         = var.instance_type
  cluster_size          = var.cluster_size
  obs_name              = var.obs_name
  obs_container_name    = var.obs_container_name
  blob_obs_access_key   = var.blob_obs_access_key
  tiering_ssd_percent   = var.tiering_ssd_percent
  subscription_id       = var.subscription_id
  private_dns_zone_name = module.create-network.private-dns-zone-name
  depends_on            = [module.create-network]
}
