resource "azurerm_log_analytics_workspace" "la_workspace" {
  name                = "${var.prefix}-${var.cluster_name}-workspace"
  location            = data.azurerm_resource_group.rg.location
  resource_group_name = data.azurerm_resource_group.rg.name
  sku                 = "PerGB2018"
  retention_in_days   = 30
}

resource "azurerm_application_insights" "application_insights" {
  name                = "${var.prefix}-${var.cluster_name}-application-insights"
  location            = data.azurerm_resource_group.rg.location
  resource_group_name = data.azurerm_resource_group.rg.name
  workspace_id        = azurerm_log_analytics_workspace.la_workspace.id
  application_type    = "web"
}
resource "azurerm_monitor_diagnostic_setting" "insights_diagnostic_setting" {
  name                       = "${var.prefix}-${var.cluster_name}-insights-diagnostic-setting"
  target_resource_id         = azurerm_application_insights.application_insights.id
  storage_account_id         = azurerm_storage_account.deployment_sa.id
  log_analytics_workspace_id = azurerm_log_analytics_workspace.la_workspace.id
  enabled_log {
    category = "AppTraces"

    retention_policy {
      enabled = false
    }
  }
  lifecycle {
    ignore_changes = [metric,log_analytics_destination_type]
  }
  depends_on = [azurerm_linux_function_app.function_app,azurerm_log_analytics_workspace.la_workspace]
}

resource "azurerm_monitor_diagnostic_setting" "function_diagnostic_setting" {
  name                       = "${var.prefix}-${var.cluster_name}-function-diagnostic-setting"
  target_resource_id         = azurerm_linux_function_app.function_app.id
  storage_account_id         = azurerm_storage_account.deployment_sa.id
  log_analytics_workspace_id = azurerm_log_analytics_workspace.la_workspace.id
  enabled_log {
    category = "FunctionAppLogs"

    retention_policy {
      enabled = false
    }
  }
  lifecycle {
    ignore_changes = [metric,log_analytics_destination_type]
  }
  depends_on = [azurerm_linux_function_app.function_app,azurerm_log_analytics_workspace.la_workspace]
}


resource "azurerm_service_plan" "app_service_plan" {
  name                = "${var.prefix}-${var.cluster_name}-app-service-plan"
  resource_group_name = data.azurerm_resource_group.rg.name
  location            = data.azurerm_resource_group.rg.location
  os_type             = "Linux"
  sku_name            = "EP2"
}

locals {
  stripe_width_calculated = var.cluster_size - var.protection_level - 1
  stripe_width = local.stripe_width_calculated < 16 ? local.stripe_width_calculated : 16
}

locals {
  location              = data.azurerm_resource_group.rg.location
  function_app_zip_name = "${var.function_app_dist}/${var.function_app_version}.zip"
  weka_sa               = "${var.function_app_storage_account_prefix}${local.location}"
  weka_sa_container     = "${var.function_app_storage_account_container_prefix}${local.location}"
}

locals {
  function_code_path     = "${path.module}/function-app/code"
  function_app_code_hash = md5(join("", [for f in fileset(local.function_code_path, "**") : filemd5("${local.function_code_path}/${f}")]))
}


resource "azurerm_linux_function_app" "function_app" {
  name                       = "${local.alphanumeric_prefix_name}-${local.alphanumeric_cluster_name}-function-app"
  resource_group_name        = data.azurerm_resource_group.rg.name
  location                   = data.azurerm_resource_group.rg.location
  service_plan_id            = azurerm_service_plan.app_service_plan.id
  storage_account_name       = azurerm_storage_account.deployment_sa.name
  storage_account_access_key = azurerm_storage_account.deployment_sa.primary_access_key
  https_only                 = true
  virtual_network_subnet_id  = var.subnet_delegation_id
  site_config {
    vnet_route_all_enabled = true
  }

  app_settings = {
    "FUNCTIONS_WORKER_RUNTIME" = "custom"
    "APPINSIGHTS_INSTRUMENTATIONKEY" = azurerm_application_insights.application_insights.instrumentation_key
    "STATE_STORAGE_NAME" = azurerm_storage_account.deployment_sa.name
    "STATE_CONTAINER_NAME" = azurerm_storage_container.deployment.name
    "HOSTS_NUM" = var.cluster_size
    "CLUSTER_NAME" = var.cluster_name
    "PROTECTION_LEVEL" = var.protection_level
    "STRIPE_WIDTH" = var.stripe_width != -1 ? var.stripe_width : local.stripe_width
    "HOTSPARE" = var.hotspare
    "VM_USERNAME" = var.vm_username
    "COMPUTE_MEMORY" = var.container_number_map[var.instance_type].memory
    "SUBSCRIPTION_ID" = data.azurerm_subscription.primary.subscription_id
    "RESOURCE_GROUP_NAME" = data.azurerm_resource_group.rg.name
    "LOCATION" = data.azurerm_resource_group.rg.location
    "SET_OBS" = var.set_obs_integration
    "OBS_NAME" = var.obs_name != "" ? var.obs_name : "${var.prefix}${var.cluster_name}obs"
    "OBS_CONTAINER_NAME" = var.obs_container_name != "" ? var.obs_container_name : "${var.prefix}-${var.cluster_name}-obs"
    "OBS_ACCESS_KEY" = var.blob_obs_access_key
    "NUM_DRIVE_CONTAINERS" = var.container_number_map[var.instance_type].drive
    "NUM_COMPUTE_CONTAINERS" = var.container_number_map[var.instance_type].compute
    "NUM_FRONTEND_CONTAINERS" = var.container_number_map[var.instance_type].frontend
    "NVMES_NUM" = var.container_number_map[var.instance_type].nvme
    "TIERING_SSD_PERCENT" = var.tiering_ssd_percent
    "PREFIX" = var.prefix
    "KEY_VAULT_URI" = azurerm_key_vault.key_vault.vault_uri
    "INSTANCE_TYPE" = var.instance_type
    "INSTALL_DPDK" = var.install_cluster_dpdk
    "NICS_NUM" = var.container_number_map[var.instance_type].nics
    "INSTALL_URL" =  var.install_weka_url != "" ? var.install_weka_url : "https://$TOKEN@get.prod.weka.io/dist/v1/install/4.2.1-09ec0d9afa5c8c98bc509df031ac03e6/4.2.1.10683-b6e17d9ed5be83e40e43ce9e2c431689"
    "LOG_LEVEL" = var.function_app_log_level

    https_only = true
    FUNCTIONS_WORKER_RUNTIME = "custom"
    FUNCTION_APP_EDIT_MODE   = "readonly"
    HASH                     = var.function_app_version
    WEBSITE_RUN_FROM_PACKAGE = "https://${local.weka_sa}.blob.core.windows.net/${local.weka_sa_container}/${local.function_app_zip_name}"
    WEBSITE_VNET_ROUTE_ALL   = true
  }

  identity {
    type = "SystemAssigned"
  }

  lifecycle {
    precondition {
      condition     = var.function_app_version == local.function_app_code_hash
      error_message = "Please update function app code version."
    }
  }

  depends_on = [azurerm_storage_account.deployment_sa]
}

data "azurerm_subscription" "primary" {}

resource "azurerm_role_assignment" "function-assignment" {
  scope                = data.azurerm_resource_group.rg.id
  role_definition_name = "Storage Blob Data Contributor"
  principal_id         = azurerm_linux_function_app.function_app.identity[0].principal_id
  depends_on           = [azurerm_linux_function_app.function_app]
}

resource "azurerm_role_assignment" "storage-blob-data-owner" {
  scope                = azurerm_storage_account.deployment_sa.id
  role_definition_name = "Storage Blob Data Owner"
  principal_id         = azurerm_linux_function_app.function_app.identity[0].principal_id
  depends_on           = [azurerm_linux_function_app.function_app, azurerm_storage_account.deployment_sa]
}

resource "azurerm_role_assignment" "storage-account-contributor" {
  scope                = data.azurerm_resource_group.rg.id
  role_definition_name = "Storage Account Contributor"
  principal_id         = azurerm_linux_function_app.function_app.identity[0].principal_id
  depends_on           = [azurerm_linux_function_app.function_app]
}

resource "azurerm_role_assignment" "function-app-key-vault-secrets-user" {
  scope                = data.azurerm_resource_group.rg.id
  role_definition_name = "Key Vault Secrets User"
  principal_id         = azurerm_linux_function_app.function_app.identity[0].principal_id
  depends_on           = [azurerm_linux_function_app.function_app]
}

resource "azurerm_role_assignment" "function-app-key-user-access-admin" {
  scope                = data.azurerm_resource_group.rg.id
  role_definition_name = "User Access Administrator"
  principal_id         = azurerm_linux_function_app.function_app.identity[0].principal_id
  depends_on           = [azurerm_linux_function_app.function_app]
}

resource "azurerm_role_assignment" "function-app-reader" {
  scope                = data.azurerm_resource_group.rg.id
  role_definition_name = "Reader"
  principal_id         = azurerm_linux_function_app.function_app.identity[0].principal_id
  depends_on           = [azurerm_linux_function_app.function_app]
}
resource "azurerm_role_assignment" "function-app-scale-set-machine-owner" {
  scope                = var.custom_image_id != null ? azurerm_linux_virtual_machine_scale_set.custom_image_vmss[0].id : azurerm_linux_virtual_machine_scale_set.default_image_vmss[0].id
  role_definition_name = "Owner"
  principal_id         = azurerm_linux_function_app.function_app.identity[0].principal_id
  depends_on           = [azurerm_linux_function_app.function_app, azurerm_linux_virtual_machine_scale_set.custom_image_vmss, azurerm_linux_virtual_machine_scale_set.default_image_vmss]
}
