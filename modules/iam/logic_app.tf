data "azurerm_user_assigned_identity" "logic_app" {
  count               = var.logic_app_identity_name != "" && var.support_logic_app ? 1 : 0
  name                = var.logic_app_identity_name
  resource_group_name = data.azurerm_resource_group.rg.name
}

resource "azurerm_user_assigned_identity" "logic_app" {
  count               = var.logic_app_identity_name == "" && var.support_logic_app ? 1 : 0
  location            = data.azurerm_resource_group.rg.location
  name                = "${var.prefix}-${var.cluster_name}-logic-app-identity"
  resource_group_name = data.azurerm_resource_group.rg.name
}

resource "azurerm_role_assignment" "logic_app_standard_reader" {
  count                = var.logic_app_identity_name == "" && var.support_logic_app ? 1 : 0
  scope                = data.azurerm_resource_group.rg.id
  role_definition_name = "Reader"
  principal_id         = azurerm_user_assigned_identity.logic_app[0].principal_id
}

resource "azurerm_role_assignment" "logic_app_standard_reader_secret" {
  count                = var.logic_app_identity_name == "" && var.support_logic_app ? 1 : 0
  scope                = var.key_vault_id
  role_definition_name = "Key Vault Secrets User"
  principal_id         = azurerm_user_assigned_identity.logic_app[0].principal_id
}

resource "azurerm_role_assignment" "logic_app_standard_reader_smb_data" {
  count                = var.logic_app_identity_name == "" && var.support_logic_app ? 1 : 0
  scope                = var.logic_app_storage_account_id
  role_definition_name = "Storage File Data SMB Share Contributor"
  principal_id         = azurerm_user_assigned_identity.logic_app[0].principal_id
}
