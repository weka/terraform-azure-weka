data "azurerm_user_assigned_identity" "vmss" {
  count               = var.vmss_identity_name != "" ? 1 : 0
  name                = var.vmss_identity_name
  resource_group_name = var.rg_name
}

resource "azurerm_user_assigned_identity" "vmss" {
  count               = var.vmss_identity_name == "" ? 1 : 0
  location            = data.azurerm_resource_group.rg.location
  name                = "${var.prefix}-${var.cluster_name}-vmss-identity"
  resource_group_name = data.azurerm_resource_group.rg.name
}

resource "azurerm_role_assignment" "reader" {
  count                = var.vmss_identity_name == "" ? 1 : 0
  scope                = data.azurerm_resource_group.rg.id
  role_definition_name = "Reader"
  principal_id         = azurerm_user_assigned_identity.vmss[0].principal_id
}

resource "azurerm_role_assignment" "network_contributor" {
  count                = var.vmss_identity_name == "" ? 1 : 0
  scope                = data.azurerm_resource_group.rg.id
  role_definition_name = "Network Contributor"
  principal_id         = azurerm_user_assigned_identity.vmss[0].principal_id
}

resource "azurerm_role_assignment" "weka_tar_data_reader" {
  count                = var.vmss_identity_name == "" && var.weka_tar_storage_account_id != "" ? 1 : 0
  scope                = var.weka_tar_storage_account_id
  role_definition_name = "Storage Blob Data Reader"
  principal_id         = azurerm_user_assigned_identity.vmss[0].principal_id
}

resource "azurerm_role_assignment" "obs_data_contributor" {
  count                = var.vmss_identity_name == "" && var.tiering_enable_obs_integration ? 1 : 0
  scope                = local.deployment_storage_account_scope
  role_definition_name = "Storage Blob Data Contributor"
  principal_id         = azurerm_user_assigned_identity.vmss[0].principal_id
}
