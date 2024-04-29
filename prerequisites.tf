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

module "iam" {
  source                         = "./modules/iam"
  rg_name                        = var.rg_name
  prefix                         = var.prefix
  cluster_name                   = var.cluster_name
  vmss_identity_name             = var.vmss_identity_name
  function_app_identity_name     = var.function_app_identity_name
  logic_app_identity_name        = var.logic_app_identity_name
  logic_app_storage_account_id   = azurerm_storage_account.logicapp.id
  key_vault_id                   = azurerm_key_vault.key_vault.id
  weka_tar_storage_account_id    = var.weka_tar_storage_account_id
  deployment_storage_account_id  = local.deployment_storage_account_id
  deployment_container_name      = local.deployment_container_name
  tiering_enable_obs_integration = var.tiering_enable_obs_integration
  tiering_obs_name               = var.tiering_obs_name
  obs_container_name             = local.obs_container_name
}

locals {
  vnet_name             = var.vnet_name == "" ? module.network.vnet_name : var.vnet_name
  vnet_rg_name          = var.vnet_rg_name == "" ? module.network.vnet_rg_name : var.vnet_rg_name
  subnet_name           = var.subnet_name == "" ? module.network.subnet_name : var.subnet_name
  sg_id                 = var.sg_id == "" ? module.network.sg_id : var.sg_id
  private_dns_zone_name = var.private_dns_zone_name == "" ? module.network.private_dns_zone_name : var.private_dns_zone_name
  private_dns_rg_name   = var.private_dns_rg_name == "" ? module.network.private_dns_rg_name : var.private_dns_rg_name
  assign_public_ip      = var.assign_public_ip != "auto" ? var.assign_public_ip == "true" : var.subnet_name == ""
  # managed identities outputs
  logic_app_identity_id           = module.iam.logic_app_identity_id
  logic_app_identity_principal    = module.iam.logic_app_identity_principal_id
  function_app_identity_id        = module.iam.function_app_identity_id
  function_app_identity_principal = module.iam.function_app_identity_principal_id
  function_app_identity_client_id = module.iam.function_app_identity_client_id
  vmss_identity_id                = module.iam.vmss_identity_id
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
