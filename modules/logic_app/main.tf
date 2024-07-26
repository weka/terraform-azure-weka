data "azurerm_client_config" "current" {}

data "azurerm_storage_account" "logicapp" {
  name                = var.storage_account_name
  resource_group_name = var.rg_name
}

resource "azurerm_service_plan" "logicapp_service_plan" {
  name                = "${var.prefix}-${var.cluster_name}-logic-app-service-plan"
  location            = var.location
  resource_group_name = var.rg_name
  os_type             = "Windows"
  sku_name            = "WS1"
  lifecycle {
    ignore_changes = all
  }
}

resource "azurerm_logic_app_standard" "logic_app_standard" {
  name                       = "${var.prefix}-${var.cluster_name}-logic-app"
  location                   = var.location
  resource_group_name        = var.rg_name
  app_service_plan_id        = azurerm_service_plan.logicapp_service_plan.id
  storage_account_name       = data.azurerm_storage_account.logicapp.name
  storage_account_access_key = data.azurerm_storage_account.logicapp.primary_access_key
  version                    = "~4" # sets FUNCTIONS_EXTENSION_VERSION (should be same as for function app)
  identity {
    type         = "UserAssigned"
    identity_ids = [var.logic_app_identity_id]
  }

  site_config {
    vnet_route_all_enabled = true
    dynamic "ip_restriction" {
      for_each = range(var.restricted_inbound_access ? 1 : 0)
      content {
        virtual_network_subnet_id = var.subnet_id
        action                    = "Allow"
        priority                  = 300
        name                      = "VirtualNetwork"
      }
    }
  }
  app_settings = {
    "FUNCTIONS_WORKER_RUNTIME"     = "node"
    "WEBSITE_NODE_DEFAULT_VERSION" = "~18"
    "function_app_key"             = var.function_app_key
    "keyVaultUri"                  = var.key_vault_uri
  }
  virtual_network_subnet_id = var.logic_app_subnet_delegation_id
  depends_on                = [azurerm_service_plan.logicapp_service_plan]
}

resource "azurerm_key_vault_access_policy" "standard_logic_app_get_secret_permission" {
  key_vault_id = var.key_vault_id
  tenant_id    = data.azurerm_client_config.current.tenant_id
  object_id    = var.logic_app_identity_principal
  secret_permissions = [
    "Get",
  ]
}


resource "azurerm_storage_share_directory" "share_directory_scale_down" {
  name                 = "site/wwwroot/scale-down"
  share_name           = "${azurerm_logic_app_standard.logic_app_standard.name}-content"
  storage_account_name = data.azurerm_storage_account.logicapp.name
}

resource "azurerm_storage_share_directory" "share_directory_scale_up" {
  name                 = "site/wwwroot/scale-up"
  share_name           = "${azurerm_logic_app_standard.logic_app_standard.name}-content"
  storage_account_name = data.azurerm_storage_account.logicapp.name
}

data "azurerm_storage_share" "storage_share" {
  name                 = "${azurerm_logic_app_standard.logic_app_standard.name}-content"
  storage_account_name = data.azurerm_storage_account.logicapp.name
}

locals {
  connections_workflow_path = "${path.module}/connections.json"
  connections_workflow = templatefile(local.connections_workflow_path, {
    function_name = var.function_app_name
    function_id   = var.function_app_id
  })
  connections_workflow_hash     = md5(join("", [for f in fileset(local.connections_workflow, "**") : filemd5("${local.connections_workflow}/${f}")]))
  connections_workflow_filename = "/tmp/${var.prefix}_${var.cluster_name}_connections_workflow_${local.connections_workflow_hash}"
  scale_up_workflow_path        = "${path.module}/scale_up.json"
  scale_up_workflow_hash        = md5(join("", [for f in fileset(local.scale_up_workflow_path, "**") : filemd5("${local.scale_up_workflow_path}/${f}")]))
  scale_up_workflow_filename    = "/tmp/${var.prefix}_${var.cluster_name}_scale_up_workflow_${local.scale_up_workflow_hash}"
  scale_down_workflow_path      = "${path.module}/scale_down.json"
  scale_down_workflow_hash      = md5(join("", [for f in fileset(local.scale_down_workflow_path, "**") : filemd5("${local.scale_down_workflow_path}/${f}")]))
  scale_down_workflow_filename  = "/tmp/${var.prefix}_${var.cluster_name}_scale_down_workflow_${local.scale_down_workflow_hash}"
}

resource "local_file" "connections_workflow_file" {
  content  = local.connections_workflow
  filename = local.connections_workflow_filename
}

resource "local_file" "scale_up_workflow_file" {
  content  = file(local.scale_up_workflow_path)
  filename = local.scale_up_workflow_filename
}

resource "local_file" "scale_down_workflow_file" {
  content  = file(local.scale_down_workflow_path)
  filename = local.scale_down_workflow_filename
}

resource "azurerm_storage_share_file" "scale_down_share_file" {
  name             = "workflow.json"
  path             = azurerm_storage_share_directory.share_directory_scale_down.name
  storage_share_id = data.azurerm_storage_share.storage_share.id
  source           = local_file.scale_down_workflow_file.filename
  depends_on       = [azurerm_storage_share_directory.share_directory_scale_down, data.azurerm_storage_share.storage_share, local_file.scale_down_workflow_file]
}

resource "azurerm_storage_share_file" "scale_up_share_file" {
  name             = "workflow.json"
  path             = azurerm_storage_share_directory.share_directory_scale_up.name
  storage_share_id = data.azurerm_storage_share.storage_share.id
  source           = local_file.scale_up_workflow_file.filename
  depends_on       = [azurerm_storage_share_directory.share_directory_scale_up, data.azurerm_storage_share.storage_share, local_file.scale_up_workflow_file]
}

resource "azurerm_storage_share_file" "connections_share_file" {
  name             = "connections.json"
  path             = "site/wwwroot"
  storage_share_id = data.azurerm_storage_share.storage_share.id
  source           = local_file.connections_workflow_file.filename
  depends_on       = [azurerm_storage_share_directory.share_directory_scale_down, data.azurerm_storage_share.storage_share, local_file.connections_workflow_file]
}
