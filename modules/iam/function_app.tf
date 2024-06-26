data "azurerm_user_assigned_identity" "function_app" {
  count               = var.function_app_identity_name != "" ? 1 : 0
  name                = var.function_app_identity_name
  resource_group_name = data.azurerm_resource_group.rg.name
}

resource "azurerm_user_assigned_identity" "function_app" {
  count               = var.function_app_identity_name == "" ? 1 : 0
  location            = data.azurerm_resource_group.rg.location
  name                = "${var.prefix}-${var.cluster_name}-function-app-identity"
  resource_group_name = data.azurerm_resource_group.rg.name
}

resource "azurerm_role_assignment" "storage_blob_data_contributor" {
  count                = var.function_app_identity_name == "" ? 1 : 0
  scope                = local.deployment_storage_account_scope
  role_definition_name = "Storage Blob Data Contributor"
  principal_id         = azurerm_user_assigned_identity.function_app[0].principal_id
}

resource "azurerm_role_assignment" "nfs_storage_blob_data_contributor" {
  count                = var.function_app_identity_name == "" && var.nfs_protocol_gateways_number > 0 ? 1 : 0
  scope                = local.nfs_deployment_sa_scope
  role_definition_name = "Storage Blob Data Contributor"
  principal_id         = azurerm_user_assigned_identity.function_app[0].principal_id
}


resource "azurerm_role_assignment" "storage_account_contributor" {
  count                = var.function_app_identity_name == "" ? 1 : 0
  scope                = data.azurerm_resource_group.rg.id
  role_definition_name = "Storage Account Contributor"
  principal_id         = azurerm_user_assigned_identity.function_app[0].principal_id
}

resource "azurerm_role_assignment" "obs_storage_blob_data_contributor" {
  count                = var.tiering_obs_name != "" && var.function_app_identity_name == "" ? 1 : 0
  scope                = local.obs_scope
  role_definition_name = "Storage Blob Data Contributor"
  principal_id         = azurerm_user_assigned_identity.function_app[0].principal_id
}

resource "azurerm_role_assignment" "function_app_key_vault_secrets_user" {
  count                = var.function_app_identity_name == "" ? 1 : 0
  scope                = var.key_vault_id
  role_definition_name = "Key Vault Secrets User"
  principal_id         = azurerm_user_assigned_identity.function_app[0].principal_id
}

resource "azurerm_role_definition" "key_vault_set_secret" {
  count       = var.function_app_identity_name == "" ? 1 : 0
  name        = "${var.prefix}-${var.cluster_name}-key-vault-new-secret-writer"
  scope       = var.key_vault_id
  description = "Can create new secrets in the key vault"

  permissions {
    actions = [
      # See: https://learn.microsoft.com/en-us/azure/role-based-access-control/permissions/security#microsoftkeyvault
      "Microsoft.KeyVault/vaults/secrets/write",
    ]
    not_actions = []
  }

  assignable_scopes = [var.key_vault_id]
}

resource "azurerm_role_assignment" "key_vault_set_secret" {
  count              = var.function_app_identity_name == "" ? 1 : 0
  scope              = var.key_vault_id
  role_definition_id = azurerm_role_definition.key_vault_set_secret[0].role_definition_resource_id
  principal_id       = azurerm_user_assigned_identity.function_app[0].principal_id
}

resource "azurerm_role_assignment" "function_app_reader" {
  count                = var.function_app_identity_name == "" ? 1 : 0
  scope                = data.azurerm_resource_group.rg.id
  role_definition_name = "Reader"
  principal_id         = azurerm_user_assigned_identity.function_app[0].principal_id
}

resource "azurerm_role_assignment" "function_app_scale_set_machine_owner" {
  count                = var.function_app_identity_name == "" ? 1 : 0
  scope                = data.azurerm_resource_group.rg.id
  role_definition_name = "Virtual Machine Contributor"
  principal_id         = azurerm_user_assigned_identity.function_app[0].principal_id
}

resource "azurerm_role_assignment" "managed_identity_operator" {
  count                = var.function_app_identity_name == "" ? 1 : 0
  scope                = data.azurerm_resource_group.rg.id
  role_definition_name = "Managed Identity Operator"
  principal_id         = azurerm_user_assigned_identity.function_app[0].principal_id
}
