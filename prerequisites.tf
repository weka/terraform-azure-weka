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
  private_dns_zone_use  = var.private_dns_zone_use
  sg_id                 = var.sg_id
  create_nat_gateway    = var.create_nat_gateway
}

module "iam" {
  source                         = "./modules/iam"
  rg_name                        = var.rg_name
  prefix                         = var.prefix
  cluster_name                   = var.cluster_name
  vnet_rg_name                   = local.vnet_rg_name
  vnet_name                      = local.vnet_name
  subnet_name                    = local.subnet_name
  vmss_identity_name             = var.vmss_identity_name
  function_app_identity_name     = var.function_app_identity_name
  support_logic_app              = local.create_logic_app
  logic_app_identity_name        = var.logic_app_identity_name
  logic_app_storage_account_id   = local.create_logic_app ? azurerm_storage_account.logicapp[0].id : ""
  key_vault_id                   = azurerm_key_vault.key_vault.id
  weka_tar_storage_account_id    = var.weka_tar_storage_account_id
  deployment_storage_account_id  = local.deployment_storage_account_id
  deployment_container_name      = local.deployment_container_name
  nfs_deployment_container_name  = local.nfs_deployment_container_name
  nfs_protocol_gateways_number   = var.nfs_protocol_gateways_number
  tiering_enable_obs_integration = var.tiering_enable_obs_integration
  tiering_obs_name               = var.tiering_obs_name
  obs_container_name             = local.obs_container_name
}

locals {
  create_logic_app      = local.create_sa_resources
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

module "logic_app_subnet_delegation" {
  count           = var.logic_app_subnet_delegation_id == "" && local.create_logic_app ? 1 : 0
  source          = "./modules/subnet_delegation"
  rg_name         = local.vnet_rg_name
  vnet_name       = local.vnet_name
  prefix          = var.prefix
  cluster_name    = var.cluster_name
  cidr_range      = var.logic_app_subnet_delegation_cidr
  delegation_name = "logic-app-delegation"

  depends_on = [module.network]
}

module "function_app_subnet_delegation" {
  count           = var.function_app_subnet_delegation_id == "" ? 1 : 0
  source          = "./modules/subnet_delegation"
  rg_name         = local.vnet_rg_name
  vnet_name       = local.vnet_name
  prefix          = var.prefix
  cluster_name    = var.cluster_name
  cidr_range      = var.function_app_subnet_delegation_cidr
  delegation_name = "function-app-delegation"

  depends_on = [module.network]
}

module "logicapp" {
  count                          = local.create_logic_app ? 1 : 0
  source                         = "./modules/logic_app"
  rg_name                        = var.rg_name
  location                       = local.location
  prefix                         = var.prefix
  cluster_name                   = var.cluster_name
  logic_app_subnet_delegation_id = var.logic_app_subnet_delegation_id == "" ? module.logic_app_subnet_delegation[0].id : var.logic_app_subnet_delegation_id
  storage_account_name           = azurerm_storage_account.logicapp[0].name
  logic_app_identity_id          = local.logic_app_identity_id
  logic_app_identity_principal   = local.logic_app_identity_principal
  restricted_inbound_access      = var.function_access_restriction_enabled
  subnet_id                      = data.azurerm_subnet.subnet.id
  function_app_id                = azurerm_linux_function_app.function_app.id
  function_app_name              = azurerm_linux_function_app.function_app.name
  function_app_key               = data.azurerm_function_app_host_keys.function_keys.default_function_key
  key_vault_id                   = azurerm_key_vault.key_vault.id
  key_vault_uri                  = azurerm_key_vault.key_vault.vault_uri

  depends_on = [azurerm_storage_account.logicapp, module.logic_app_subnet_delegation, module.iam]
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
