data "azurerm_resource_group" "rg" {
  name = var.rg_name
}

module "network" {
  source                = "./modules/network"
  prefix                = var.prefix
  vnet_name             = var.vnet_name
  subnet_name           = var.subnet_name
  rg_name               = var.rg_name
  vnet_rg_name          = var.vnet_rg_name
  private_dns_rg_name   = var.private_dns_rg_name
  address_space         = var.address_space
  subnet_prefix         = var.subnet_prefix
  allow_ssh_cidrs       = var.allow_ssh_cidrs
  allow_weka_api_cidrs  = var.allow_weka_api_cidrs
  private_dns_zone_name = var.private_dns_zone_name
  sg_id                 = var.sg_id
  create_nat_gateway    = var.create_nat_gateway
}

locals {
  vnet_name             = var.vnet_name == "" ? module.network.vnet_name : var.vnet_name
  vnet_rg_name          = var.vnet_rg_name == "" ? module.network.vnet_rg_name : var.vnet_rg_name
  subnet_name           = var.subnet_name == "" ? module.network.subnet_name : var.subnet_name
  sg_id                 = var.sg_id == "" ? module.network.sg_id : var.sg_id
  private_dns_zone_name = var.private_dns_zone_name == "" ? module.network.private_dns_zone_name : var.private_dns_zone_name
  private_dns_rg_name   = var.private_dns_rg_name == "" ? module.network.private_dns_rg_name : var.private_dns_rg_name
  assign_public_ip      = var.assign_public_ip == "auto" ? var.create_nat_gateway ? false : true : var.assign_public_ip
}

module "peering" {
  count                            = length(var.vnets_to_peer_to_deployment_vnet)
  source                           = "./modules/peering_vnets"
  vnet_rg_name                     = local.vnet_rg_name
  vnet_name                        = local.vnet_name
  vnets_to_peer_to_deployment_vnet = var.vnets_to_peer_to_deployment_vnet
  depends_on                       = [module.network]
}

data "azurerm_subnet" "subnet" {
  resource_group_name  = local.vnet_rg_name
  virtual_network_name = local.vnet_name
  name                 = local.subnet_name
  depends_on           = [module.network]
}

data "azurerm_virtual_network" "vnet" {
  name                = local.vnet_name
  resource_group_name = local.vnet_rg_name
  depends_on          = [module.network]
}
