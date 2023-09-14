data "azurerm_resource_group" "rg" {
  name = var.rg_name
}

module "network" {
  count             = var.subnet_name == "" ? 1 : 0
  source            = "./modules/network"
  prefix            = var.prefix
  rg_name           = var.rg_name
  address_space     = var.address_space
  subnet_prefix     = var.subnet_prefix
  allow_ssh_ranges  = var.allow_ssh_ranges
  private_network   = var.private_network
}

locals {
  vnet_name    = var.vnet_name == "" ? module.network[0].vnet_name : var.vnet_name
  vnet_rg_name = var.vnet_rg_name == "" ? module.network[0].vnet_rg_name :  var.vnet_rg_name
  subnet_name  = var.subnet_name == "" ? module.network[0].subnet_name : var.subnet_name
  sg_id        = var.sg_id == "" ? module.network[0].sg_id : var.sg_id
  private_dns_zone_name = var.private_dns_zone_name == "" ? module.network[0].private_dns_zone_name : var.private_dns_zone_name
}

module "peering" {
  count           = length(var.vnet_to_peering) > 0 ? 1 : 0
  source          = "./modules/peering_vnets"
  prefix          = var.prefix
  rg_name         = var.rg_name
  vnet_name       = local.vnet_name
  vnet_to_peering = var.vnet_to_peering
  depends_on      = [module.network]
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

