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
  private_network   = var.private_network
  subnets_delegation_prefixes = var.subnets_delegation_prefixes
}

module "deploy-weka" {
  count                    = length(var.clusters_list)
  source                   = "../.."
  prefix                   = var.prefix
  rg_name                  = var.rg_name
  vnet_name                = module.create-network.vnet-name
  vnet_rg_name             = module.create-network.vnet_rg_name
  subnets                  = module.create-network.subnets-name
  sg_id                    = module.create-network.sg-id
  subnets_delegation_names = [module.create-network.subnets_delegation_names[count.index]]
  blob_dns_zone_name       = module.create-network.blob-dns-zone-name
  keyvault_dns_zone_name   = module.create-network.keyvault-dns-zone-name
  sites_dns_zone_name      = module.create-network.sites-dns-zone-name
  apt_repo_url             = var.apt_repo_url
  private_network          = var.private_network
  install_weka_url         = var.install_weka_url
  cluster_name             = var.clusters_list[count.index]
  install_ofed_url         = var.install_ofed_url
  instance_type            = var.instance_type
  cluster_size             = var.cluster_size
  set_obs_integration      = var.set_obs_integration
  tiering_ssd_percent      = var.tiering_ssd_percent
  subscription_id          = var.subscription_id
  private_dns_zone_name    = module.create-network.private-dns-zone-name
  depends_on               = [module.create-network]
}