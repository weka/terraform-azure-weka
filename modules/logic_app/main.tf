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

resource "azurerm_storage_share" "storage_share" {
  count                = var.use_secured_storage_account ? 1 : 0
  name                 = "${var.prefix}-${var.cluster_name}-logic-app-content"
  storage_account_name = data.azurerm_storage_account.logicapp.name
  quota                = 100
}

data "azurerm_storage_share" "storage_share" {
  count                = var.use_secured_storage_account ? 0 : 1
  name                 = "${azurerm_logic_app_standard.logic_app_standard.name}-content"
  storage_account_name = data.azurerm_storage_account.logicapp.name
}

locals {
  storage_share_id = var.use_secured_storage_account ? azurerm_storage_share.storage_share[0].id : data.azurerm_storage_share.storage_share[0].id
}

resource "azurerm_logic_app_standard" "logic_app_standard" {
  name                       = "${var.prefix}-${var.cluster_name}-logic-app"
  location                   = var.location
  resource_group_name        = var.rg_name
  app_service_plan_id        = azurerm_service_plan.logicapp_service_plan.id
  storage_account_share_name = var.use_secured_storage_account ? azurerm_storage_share.storage_share[0].name : null
  storage_account_name       = data.azurerm_storage_account.logicapp.name
  storage_account_access_key = data.azurerm_storage_account.logicapp.primary_access_key
  version                    = "~4" # sets FUNCTIONS_EXTENSION_VERSION (should be same as for function app)
  identity {
    type         = "UserAssigned"
    identity_ids = [var.logic_app_identity_id]
  }

  site_config {
    public_network_access_enabled = false
    vnet_route_all_enabled        = true
    always_on                     = true
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
    "WEBSITE_CONTENTOVERVNET"      = var.use_secured_storage_account ? 1 : 0
    "FUNCTIONS_WORKER_RUNTIME"     = "node"
    "WEBSITE_NODE_DEFAULT_VERSION" = "~18"
    "function_app_key"             = var.function_app_key
    "keyVaultUri"                  = var.key_vault_uri
  }
  https_only                = true
  virtual_network_subnet_id = var.logic_app_subnet_delegation_id

  depends_on = [azurerm_service_plan.logicapp_service_plan]
}

resource "azurerm_key_vault_access_policy" "standard_logic_app_get_secret_permission" {
  key_vault_id = var.key_vault_id
  tenant_id    = data.azurerm_client_config.current.tenant_id
  object_id    = var.logic_app_identity_principal
  secret_permissions = [
    "Get",
  ]
}

resource "null_resource" "wait_for_logic_app" {
  count = var.use_secured_storage_account ? 1 : 0
  triggers = {
    logic_app_id = azurerm_logic_app_standard.logic_app_standard.id
  }

  provisioner "local-exec" {
    # wait for "site/wwwroot" to be created in file share
    command = "sleep 60"
  }

  depends_on = [azurerm_logic_app_standard.logic_app_standard]
}

resource "azurerm_storage_share_directory" "share_directory_scale_down" {
  name             = "site/wwwroot/scale-down"
  storage_share_id = local.storage_share_id
  depends_on       = [null_resource.wait_for_logic_app]
}

resource "azurerm_storage_share_directory" "share_directory_scale_up" {
  name             = "site/wwwroot/scale-up"
  storage_share_id = local.storage_share_id
  depends_on       = [null_resource.wait_for_logic_app]
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
  storage_share_id = local.storage_share_id
  source           = local_file.scale_down_workflow_file.filename
  depends_on       = [azurerm_storage_share_directory.share_directory_scale_down, azurerm_storage_share.storage_share, local_file.scale_down_workflow_file]
}

resource "azurerm_storage_share_file" "scale_up_share_file" {
  name             = "workflow.json"
  path             = azurerm_storage_share_directory.share_directory_scale_up.name
  storage_share_id = local.storage_share_id
  source           = local_file.scale_up_workflow_file.filename
  depends_on       = [azurerm_storage_share_directory.share_directory_scale_up, azurerm_storage_share.storage_share, local_file.scale_up_workflow_file]
}

resource "azurerm_storage_share_file" "connections_share_file" {
  name             = "connections.json"
  path             = "site/wwwroot"
  storage_share_id = local.storage_share_id
  source           = local_file.connections_workflow_file.filename
  depends_on       = [azurerm_storage_share_directory.share_directory_scale_down, azurerm_storage_share.storage_share, local_file.connections_workflow_file]
}
