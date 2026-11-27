data "azurerm_client_config" "current" {}

resource "azurerm_key_vault" "key_vault" {
  name                     = "${local.alphanumeric_prefix_name}-${local.alphanumeric_cluster_name}"
  location                 = data.azurerm_resource_group.rg.location
  resource_group_name      = var.rg_name
  enabled_for_deployment   = true
  tenant_id                = data.azurerm_client_config.current.tenant_id
  purge_protection_enabled = var.key_vault_purge_protection_enabled
  sku_name                 = "standard"
  tags                     = local.tags

  # Granted in the vault's create request rather than by a separate access policy resource:
  # a policy added right after the vault can still be missing on some Key Vault replicas when
  # the secrets below are written, failing them with a 403.
  access_policy {
    tenant_id = data.azurerm_client_config.current.tenant_id
    object_id = data.azurerm_client_config.current.object_id
    key_permissions = [
      "Get", "List", "Update", "Create", "Import", "Delete", "Recover", "Backup", "Purge",
    ]
    secret_permissions = [
      "Get", "List", "Delete", "Recover", "Backup", "Restore", "Set", "Purge",
    ]
    storage_permissions = [
      "Get",
      "List",
    ]
  }

  lifecycle {
    # An inline access_policy list is authoritative; ignoring it keeps the policies other
    # resources add with azurerm_key_vault_access_policy.
    ignore_changes = [tags, access_policy]
  }
}

resource "azurerm_key_vault_access_policy" "function_app_secret_permissions" {
  key_vault_id = azurerm_key_vault.key_vault.id
  tenant_id    = data.azurerm_client_config.current.tenant_id
  object_id    = local.function_app_identity_principal

  secret_permissions = [
    "Get", "Set"
  ]

  depends_on = [azurerm_key_vault.key_vault]
}

# Deployments created before the policy moved into the vault keep it in Azure; destroying it
# would revoke the deployer's access to the secrets.
removed {
  from = azurerm_key_vault_access_policy.key_vault_access_policy
  lifecycle {
    destroy = false
  }
}

resource "azurerm_key_vault_secret" "public_ssh_keys" {
  count        = var.ssh_public_key == null ? 1 : 0
  name         = "public-key"
  value        = tls_private_key.ssh_key[0].public_key_openssh
  key_vault_id = azurerm_key_vault.key_vault.id
  tags         = local.tags
  lifecycle {
    ignore_changes = [value, tags]
  }
  depends_on = [azurerm_key_vault.key_vault, tls_private_key.ssh_key]
}

resource "azurerm_key_vault_secret" "private_ssh_keys" {
  count        = var.ssh_public_key == null ? 1 : 0
  name         = "private-key"
  value        = tls_private_key.ssh_key[0].private_key_pem
  key_vault_id = azurerm_key_vault.key_vault.id
  tags         = local.tags
  lifecycle {
    ignore_changes = [value, tags]
  }
  depends_on = [azurerm_key_vault.key_vault, tls_private_key.ssh_key]
}

data "azurerm_function_app_host_keys" "function_keys" {
  name                = azurerm_linux_function_app.function_app.name
  resource_group_name = data.azurerm_resource_group.rg.name
  depends_on          = [azurerm_linux_function_app.function_app]
}

resource "azurerm_key_vault_secret" "function_app_default_key" {
  name         = "function-app-default-key"
  value        = data.azurerm_function_app_host_keys.function_keys.default_function_key
  key_vault_id = azurerm_key_vault.key_vault.id
  tags         = local.tags
  depends_on   = [azurerm_key_vault.key_vault]
  lifecycle {
    ignore_changes = [tags]
  }
}

resource "azurerm_key_vault_secret" "get_weka_io_token" {
  name         = "get-weka-io-token"
  value        = var.get_weka_io_token
  key_vault_id = azurerm_key_vault.key_vault.id
  tags         = local.tags
  depends_on   = [azurerm_key_vault.key_vault]
  lifecycle {
    ignore_changes = [tags]
  }
}

resource "azurerm_key_vault_secret" "weka_password_secret" {
  name         = "weka-password"
  value        = ""
  key_vault_id = azurerm_key_vault.key_vault.id
  tags         = local.tags
  lifecycle {
    ignore_changes = [value, tags]
  }
  depends_on = [azurerm_key_vault.key_vault]
}


resource "azurerm_key_vault_secret" "weka_deployment_password" {
  name         = "weka-deployment-password"
  value        = ""
  key_vault_id = azurerm_key_vault.key_vault.id
  tags         = local.tags
  lifecycle {
    ignore_changes = [value, tags]
  }
  depends_on = [azurerm_key_vault.key_vault]
}
