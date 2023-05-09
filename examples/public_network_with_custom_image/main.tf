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
  subnet_prefixes   = var.subnet_prefixes
}

module "deploy-weka" {
  source                = "../.."
  prefix                = var.prefix
  rg_name               = var.rg_name
  vnet_name             = module.create-network.vnet-name
  subnets               = module.create-network.subnets-name
  sg_id                 = module.create-network.sg-id
  get_weka_io_token     = var.get_weka_io_token
  subnet_delegation     = var.subnet_delegation
  cluster_name          = var.cluster_name
  set_obs_integration   = var.set_obs_integration
  instance_type         = var.instance_type
  cluster_size          = var.cluster_size
  install_ofed          = var.install_ofed
  tiering_ssd_percent   = var.tiering_ssd_percent
  subscription_id       = var.subscription_id
  custom_image_id       = var.custom_image_id
  private_dns_zone_name = module.create-network.private-dns-zone-name
  depends_on            = [module.create-network]
}
