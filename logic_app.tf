resource "azurerm_storage_account" "logicapp" {
  name                     = "${var.prefix}${var.cluster_name}logicappsa"
  resource_group_name      = var.rg_name
  location                 = local.location
  account_tier             = "Standard"
  account_replication_type = "LRS"
}

resource "azurerm_subnet" "logicapp_subnet_delegation" {
  count                = var.logicapp_subnet_delegation_id == "" ? 1 : 0
  name                 = "${var.prefix}-${var.cluster_name}-logicapp-delegation"
  resource_group_name  = local.vnet_rg_name
  virtual_network_name = local.vnet_name
  address_prefixes     = [var.logicapp_subnet_delegation_cdir]
  service_endpoints    = ["Microsoft.KeyVault","Microsoft.Web"]
  delegation {
    name = "logic-delegation"
    service_delegation {
      name    = "Microsoft.Web/serverFarms"
      actions = ["Microsoft.Network/virtualNetworks/subnets/action"]
    }
  }
}

resource "azurerm_service_plan" "logicapp_service_plan" {
  name                = "${var.prefix}-${var.cluster_name}-logic-app-service-plan"
  location            = local.location
  resource_group_name = var.rg_name
  os_type             = "Windows"
  sku_name            = "WS1"
  lifecycle {
    ignore_changes = all
  }
}

resource "azurerm_logic_app_standard" "logic_app_standard" {
  name                       = "${var.prefix}-${var.cluster_name}-logic-app"
  location                   = local.location
  resource_group_name        = var.rg_name
  app_service_plan_id        = azurerm_service_plan.logicapp_service_plan.id
  storage_account_name       = azurerm_storage_account.logicapp.name
  storage_account_access_key = azurerm_storage_account.logicapp.primary_access_key
  identity {
    type = "SystemAssigned"
  }
  site_config {
    vnet_route_all_enabled = true
    dynamic "ip_restriction" {
      for_each = range(local.create_private_function)
      content {
        virtual_network_subnet_id = data.azurerm_subnet.subnet.id
        action                    = "Allow"
        priority                  = 300
        name                      = "VirtualNetwork"
      }
    }
  }
  app_settings = {
    "FUNCTIONS_WORKER_RUNTIME"     = "node"
    "WEBSITE_NODE_DEFAULT_VERSION" = "~18"
  }
  virtual_network_subnet_id = var.logicapp_subnet_delegation_id == "" ? azurerm_subnet.logicapp_subnet_delegation[0].id : var.logicapp_subnet_delegation_id
  depends_on                = [azurerm_service_plan.logicapp_service_plan, azurerm_subnet.logicapp_subnet_delegation, azurerm_storage_account.logicapp]
  lifecycle {
    ignore_changes = [site_config]
  }
}

resource "azurerm_key_vault_access_policy" "standard_logic_app_get_secret_permission" {
  key_vault_id = azurerm_key_vault.key_vault.id
  tenant_id    = data.azurerm_client_config.current.tenant_id
  object_id    = azurerm_logic_app_standard.logic_app_standard.identity[0].principal_id
  secret_permissions = [
    "Get",
  ]
  depends_on = [azurerm_key_vault.key_vault, azurerm_logic_app_standard.logic_app_standard]
}


resource "azurerm_storage_share_directory" "share_directory_scale_down" {
  name                 = "site/wwwroot/scale_down"
  share_name           = "${azurerm_logic_app_standard.logic_app_standard.name}-content"
  storage_account_name = azurerm_storage_account.logicapp.name
  depends_on           = [azurerm_storage_account.logicapp]
}

resource "azurerm_storage_share_directory" "share_directory_scale_up" {
  name                 = "site/wwwroot/scale_up"
  share_name           = "${azurerm_logic_app_standard.logic_app_standard.name}-content"
  storage_account_name = azurerm_storage_account.logicapp.name
  depends_on           = [azurerm_storage_account.logicapp]
}

data "azurerm_storage_share" "storage_share" {
  name                 = "${azurerm_logic_app_standard.logic_app_standard.name}-content"
  storage_account_name = azurerm_storage_account.logicapp.name
}

locals {
  scale_down_workflow = templatefile("${path.module}/logic_app/scale_down.json", {
    function_name         = azurerm_linux_function_app.function_app.name
    function_app_key_name = azurerm_key_vault_secret.function_app_default_key.name
  })
  scale_up_workflow = templatefile("${path.module}/logic_app/scale_up.json", {
    function_name         = azurerm_linux_function_app.function_app.name
    function_app_key_name = azurerm_key_vault_secret.function_app_default_key.name
  })
  connections_workflow = templatefile("${path.module}/logic_app/connections.json", {
    keyvault_name = azurerm_key_vault.key_vault.name
  })
}

resource "local_file" "scale_down_workflow_file" {
  content  = local.scale_down_workflow
  filename = "/tmp/scale-down.json"
}

resource "local_file" "scale_up_workflow_file" {
  content  = local.scale_up_workflow
  filename = "/tmp/scale-up.json"
}

resource "local_file" "connections_workflow_file" {
  content  = local.connections_workflow
  filename = "/tmp/connections.json"
}

resource "azurerm_storage_share_file" "scale_down_share_file" {
  name             = "workflow.json"
  path             = "site/wwwroot/scale_down"
  storage_share_id = data.azurerm_storage_share.storage_share.id
  source           = local_file.scale_down_workflow_file.filename
  depends_on       = [azurerm_storage_share_directory.share_directory_scale_down, data.azurerm_storage_share.storage_share, local_file.scale_down_workflow_file]
}

resource "azurerm_storage_share_file" "scale_up_share_file" {
  name             = "workflow.json"
  path             = "site/wwwroot/scale_up"
  storage_share_id = data.azurerm_storage_share.storage_share.id
  source           = local_file.scale_up_workflow_file.filename
  depends_on       = [azurerm_storage_share_directory.share_directory_scale_up, data.azurerm_storage_share.storage_share, local_file.scale_up_workflow_file]
}

resource "azurerm_storage_share_file" "connections_share_file" {
  name             = "connections.json"
  path             = "site/wwwroot"
  storage_share_id = data.azurerm_storage_share.storage_share.id
  source           = local_file.connections_workflow_file.filename
  depends_on       = [azurerm_storage_share_directory.share_directory_scale_down, data.azurerm_storage_share.storage_share]
}

resource "azurerm_role_assignment" "logic_app_standard_reader" {
  scope                = data.azurerm_resource_group.rg.id
  role_definition_name = "Reader"
  principal_id         = azurerm_logic_app_standard.logic_app_standard.identity[0].principal_id
  depends_on           = [azurerm_logic_app_standard.logic_app_standard]
}

resource "azurerm_role_assignment" "logic_app_standard_reader_secret" {
  scope                = azurerm_key_vault.key_vault.id
  role_definition_name = "Key Vault Secrets User"
  principal_id         = azurerm_logic_app_standard.logic_app_standard.identity[0].principal_id
  depends_on           = [azurerm_logic_app_standard.logic_app_standard, azurerm_key_vault.key_vault]
}

resource "azurerm_role_assignment" "logic_app_standard_reader_smb_data" {
  scope                = azurerm_storage_account.logicapp.id
  role_definition_name = "Storage File Data SMB Share Contributor"
  principal_id         = azurerm_logic_app_standard.logic_app_standard.identity[0].principal_id
  depends_on           = [azurerm_logic_app_standard.logic_app_standard, azurerm_storage_account.logicapp]
}

