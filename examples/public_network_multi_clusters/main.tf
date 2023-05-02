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
  count                 = length(var.clusters_list)
  source                = "../.."
  prefix                = var.prefix
  rg_name               = var.rg_name
  vnet_name             = module.create-network.vnet-name
  subnets               = module.create-network.subnets-name
  sg_id                 = module.create-network.sg-id
  subnet_delegation_id  = module.create-network.subnet-delegation-id
  get_weka_io_token     = var.get_weka_io_token
  instance_type         = var.instance_type
  cluster_size          = var.cluster_size
  set_obs_integration   = var.set_obs_integration
  tiering_ssd_percent   = var.tiering_ssd_percent
  cluster_name          = var.clusters_list[count.index]
  subscription_id       = var.subscription_id
  install_ofed = false
  custom_image_id = "/subscriptions/d2f248b9-d054-477f-b7e8-413921532c2a/resourceGroups/weka-tf/providers/Microsoft.Compute/images/weka-custome-image-ofed-5.6-image"
  private_dns_zone_name = module.create-network.private-dns-zone-name
  depends_on            = [module.create-network]
}