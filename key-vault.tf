data "azurerm_client_config" "current" {}
locals {
  create_private_key_vault = var.create_public_key_vault ? 0 : 1
}
resource "azurerm_key_vault" "key_vault" {
  name                     = "${local.alphanumeric_prefix_name}-${local.alphanumeric_cluster_name}"
  location                 = data.azurerm_resource_group.rg.location
  resource_group_name      = var.rg_name
  enabled_for_deployment   = true
  tenant_id                = data.azurerm_client_config.current.tenant_id
  purge_protection_enabled = false
  sku_name                 = "standard"
  tags                     = merge(var.tags_map, { "weka_cluster" : var.cluster_name })
  lifecycle {
    ignore_changes = [tags]
  }
}

resource "azurerm_key_vault_access_policy" "function_app_get_secret_permission" {
  key_vault_id = azurerm_key_vault.key_vault.id
  tenant_id    = data.azurerm_client_config.current.tenant_id
  object_id    = local.function_app_identity_principal

  secret_permissions = [
    "Get",
  ]

  depends_on = [azurerm_key_vault.key_vault]
}

resource "azurerm_key_vault_access_policy" "key_vault_access_policy" {
  tenant_id    = data.azurerm_client_config.current.tenant_id
  object_id    = data.azurerm_client_config.current.object_id
  key_vault_id = azurerm_key_vault.key_vault.id
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

  depends_on = [azurerm_key_vault.key_vault]
}

resource "azurerm_key_vault_secret" "public_ssh_keys" {
  count        = var.ssh_public_key == null ? 1 : 0
  name         = "public-key"
  value        = tls_private_key.ssh_key[0].public_key_openssh
  key_vault_id = azurerm_key_vault.key_vault.id
  tags         = merge(var.tags_map, { "weka_cluster" : var.cluster_name })
  lifecycle {
    ignore_changes = [value, tags]
  }
  depends_on = [azurerm_key_vault.key_vault, tls_private_key.ssh_key, azurerm_key_vault_access_policy.key_vault_access_policy]
}

resource "azurerm_key_vault_secret" "private_ssh_keys" {
  count        = var.ssh_public_key == null ? 1 : 0
  name         = "private-key"
  value        = tls_private_key.ssh_key[0].private_key_pem
  key_vault_id = azurerm_key_vault.key_vault.id
  tags         = merge(var.tags_map, { "weka_cluster" : var.cluster_name })
  lifecycle {
    ignore_changes = [value, tags]
  }
  depends_on = [azurerm_key_vault.key_vault, tls_private_key.ssh_key, azurerm_key_vault_access_policy.key_vault_access_policy]
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
  tags         = merge(var.tags_map, { "weka_cluster" : var.cluster_name })
  depends_on   = [azurerm_key_vault.key_vault, azurerm_key_vault_access_policy.key_vault_access_policy]
  lifecycle {
    ignore_changes = [tags]
  }
}

resource "azurerm_key_vault_secret" "get_weka_io_token" {
  name         = "get-weka-io-token"
  value        = var.get_weka_io_token
  key_vault_id = azurerm_key_vault.key_vault.id
  tags         = merge(var.tags_map, { "weka_cluster" : var.cluster_name })
  depends_on   = [azurerm_key_vault.key_vault, azurerm_key_vault_access_policy.key_vault_access_policy]
  lifecycle {
    ignore_changes = [tags]
  }
}

resource "random_password" "weka_password" {
  length      = 16
  lower       = true
  min_lower   = 1
  upper       = true
  min_upper   = 1
  numeric     = true
  min_numeric = 1
}

resource "azurerm_key_vault_secret" "weka_password_secret" {
  name         = "weka-password"
  value        = random_password.weka_password.result
  key_vault_id = azurerm_key_vault.key_vault.id
  tags         = merge(var.tags_map, { "weka_cluster" : var.cluster_name })
  lifecycle {
    ignore_changes = [value, tags]
  }
  depends_on = [azurerm_key_vault.key_vault, random_password.weka_password, azurerm_key_vault_access_policy.key_vault_access_policy]
}

resource azurerm_private_dns_zone "keyvault_dns" {
  name                = "privatelink.vaultcore.azure.net"
  resource_group_name = var.rg_name
}

resource "azurerm_private_dns_zone_virtual_network_link" "kv_private_link" {
  name                  = "keyvault_privatelink"
  resource_group_name   = var.rg_name
  private_dns_zone_name = azurerm_private_dns_zone.keyvault_dns.name
  virtual_network_id    = data.azurerm_virtual_network.vnet.id
}

resource "azurerm_private_endpoint" "key_vault_endpoint" {
  name                          = "${var.prefix}-${var.cluster_name}-key-vault-endpoint"
  resource_group_name           = var.rg_name
  location                      = data.azurerm_resource_group.rg.location
  subnet_id                     = data.azurerm_subnet.subnet.id
  custom_network_interface_name = "${var.prefix}-${var.cluster_name}-key-vault-endpoint"
  tags                          = merge(var.tags_map, {"weka_cluster": var.cluster_name})
  private_service_connection {
    name                           = "${var.prefix}-${var.cluster_name}-key-vault-endpoint"
    private_connection_resource_id = azurerm_key_vault.key_vault.id
    is_manual_connection           = false
    subresource_names              = ["vault"]
  }
  private_dns_zone_group {
    name                 = "keyvault-private-dns-zone-group"
    private_dns_zone_ids = [azurerm_private_dns_zone.keyvault_dns.id]
  }

  lifecycle {
    ignore_changes = [tags]
  }
  depends_on = [azurerm_key_vault.key_vault]
}

