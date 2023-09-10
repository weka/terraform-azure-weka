locals {
  stripe_width_calculated          = var.cluster_size - var.protection_level - 1
  stripe_width                     = local.stripe_width_calculated < 16 ? local.stripe_width_calculated : 16
  location                         = data.azurerm_resource_group.rg.location
  function_app_zip_name            = "${var.function_app_dist}/${var.function_app_version}.zip"
  weka_sa                          = "${var.function_app_storage_account_prefix}${local.location}"
  weka_sa_container                = "${var.function_app_storage_account_container_prefix}${local.location}"
  function_code_path               = "${path.module}/function-app/code"
  function_app_code_hash           = md5(join("", [for f in fileset(local.function_code_path, "**") : filemd5("${local.function_code_path}/${f}")]))
  get_compute_memory_index         = var.add_frontend_containers ? 1 : 0
  deployment_storage_account_id    = var.deployment_storage_account_name == "" ? azurerm_storage_account.deployment_sa[0].id : data.azurerm_storage_account.deployment_blob[0].id
  deployment_storage_account_name  = var.deployment_storage_account_name == "" ? azurerm_storage_account.deployment_sa[0].name : var.deployment_storage_account_name
  deployment_container_name        = var.deployment_container_name == "" ? azurerm_storage_container.deployment[0].name : var.deployment_container_name
  deployment_storage_account_scope = "${local.deployment_storage_account_id}/blobServices/default/containers/${local.deployment_container_name}"
  obs_storage_account_name         = var.obs_name == "" ? "${local.alphanumeric_prefix_name}${local.alphanumeric_cluster_name}obs" : var.obs_name
  obs_container_name               = var.obs_container_name == "" ? "${var.prefix}-${var.cluster_name}-obs" : var.obs_container_name
  obs_id                           = var.obs_name != "" ? data.azurerm_storage_account.obs_sa[0].id : ""
  obs_scope                        = var.obs_name != "" ? "${data.azurerm_storage_account.obs_sa[0].id}/blobServices/default/containers/${local.obs_container_name}" : ""
  function_app_name                = "${local.alphanumeric_prefix_name}-${local.alphanumeric_cluster_name}-function-app"
}

resource "azurerm_log_analytics_workspace" "la_workspace" {
  name                = "${local.alphanumeric_prefix_name}-${local.alphanumeric_cluster_name}-workspace"
  location            = data.azurerm_resource_group.rg.location
  resource_group_name = data.azurerm_resource_group.rg.name
  sku                 = "PerGB2018"
  retention_in_days   = 30
  lifecycle {
    ignore_changes = [tags]
  }
}

resource "azurerm_application_insights" "application_insights" {
  name                = "${var.prefix}-${var.cluster_name}-application-insights"
  location            = data.azurerm_resource_group.rg.location
  resource_group_name = data.azurerm_resource_group.rg.name
  workspace_id        = azurerm_log_analytics_workspace.la_workspace.id
  application_type    = "web"
  lifecycle {
    ignore_changes = [tags]
  }
}

resource "azurerm_monitor_diagnostic_setting" "insights_diagnostic_setting" {
  name                       = "${var.prefix}-${var.cluster_name}-insights-diagnostic-setting"
  target_resource_id         = azurerm_application_insights.application_insights.id
  storage_account_id         = local.deployment_storage_account_id
  log_analytics_workspace_id = azurerm_log_analytics_workspace.la_workspace.id
  enabled_log {
    category = "AppTraces"

    retention_policy {
      enabled = false
    }
  }
  lifecycle {
    ignore_changes = [metric, log_analytics_destination_type]
  }
  depends_on = [azurerm_linux_function_app.function_app, azurerm_log_analytics_workspace.la_workspace]
}

resource "azurerm_monitor_diagnostic_setting" "function_diagnostic_setting" {
  name                       = "${var.prefix}-${var.cluster_name}-function-diagnostic-setting"
  target_resource_id         = azurerm_linux_function_app.function_app.id
  storage_account_id         = local.deployment_storage_account_id
  log_analytics_workspace_id = azurerm_log_analytics_workspace.la_workspace.id
  enabled_log {
    category = "FunctionAppLogs"

    retention_policy {
      enabled = false
    }
  }
  lifecycle {
    ignore_changes = [metric, log_analytics_destination_type]
  }
  depends_on = [azurerm_linux_function_app.function_app, azurerm_log_analytics_workspace.la_workspace]
}

resource "azurerm_subnet" "subnet_delegation" {
  count                = var.subnet_delegation_id == null ? 1 : 0
  name                 = "${var.prefix}-${var.cluster_name}-subnet-delegation"
  resource_group_name  = var.rg_name
  virtual_network_name = data.azurerm_virtual_network.vnet.name
  address_prefixes     = [var.subnet_delegation]

  delegation {
    name = "subnet-delegation"
    service_delegation {
      name    = "Microsoft.Web/serverFarms"
      actions = ["Microsoft.Network/virtualNetworks/subnets/action"]
    }
  }
  depends_on = [module.network]
}

resource "azurerm_service_plan" "app_service_plan" {
  name                = "${var.prefix}-${var.cluster_name}-app-service-plan"
  resource_group_name = data.azurerm_resource_group.rg.name
  location            = data.azurerm_resource_group.rg.location
  os_type             = "Linux"
  sku_name            = "EP1"
  lifecycle {
    ignore_changes = [tags]
  }
}

resource "azurerm_linux_function_app" "function_app" {
  name                       = local.function_app_name
  resource_group_name        = data.azurerm_resource_group.rg.name
  location                   = data.azurerm_resource_group.rg.location
  service_plan_id            = azurerm_service_plan.app_service_plan.id
  storage_account_name       = local.deployment_storage_account_name
  storage_account_access_key = var.deployment_storage_account_access_key == "" ? azurerm_storage_account.deployment_sa[0].primary_access_key : var.deployment_storage_account_access_key
  https_only                 = true
  virtual_network_subnet_id  = var.subnet_delegation_id == null ? azurerm_subnet.subnet_delegation[0].id : var.subnet_delegation_id
  site_config {
    vnet_route_all_enabled = true
  }

  app_settings = {
    "APPINSIGHTS_INSTRUMENTATIONKEY" = azurerm_application_insights.application_insights.instrumentation_key
    "STATE_STORAGE_NAME"             = local.deployment_storage_account_name
    "STATE_CONTAINER_NAME"           = local.deployment_container_name
    "HOSTS_NUM"                      = var.cluster_size
    "CLUSTER_NAME"                   = var.cluster_name
    "PROTECTION_LEVEL"               = var.protection_level
    "STRIPE_WIDTH"                   = var.stripe_width != -1 ? var.stripe_width : local.stripe_width
    "HOTSPARE"                       = var.hotspare
    "VM_USERNAME"                    = var.vm_username
    "SUBSCRIPTION_ID"                = data.azurerm_subscription.primary.subscription_id
    "RESOURCE_GROUP_NAME"            = data.azurerm_resource_group.rg.name
    "LOCATION"                       = data.azurerm_resource_group.rg.location
    "SET_OBS"                        = var.set_obs_integration
    "OBS_NAME"                       = local.obs_storage_account_name
    "OBS_CONTAINER_NAME"             = local.obs_container_name
    "OBS_ACCESS_KEY"                 = var.blob_obs_access_key
    NUM_DRIVE_CONTAINERS             = var.container_number_map[var.instance_type].drive
    NUM_COMPUTE_CONTAINERS           = var.add_frontend_containers == false ? var.container_number_map[var.instance_type].compute + 1 : var.container_number_map[var.instance_type].compute
    NUM_FRONTEND_CONTAINERS          = var.add_frontend_containers == false ? 0 : var.container_number_map[var.instance_type].frontend
    COMPUTE_MEMORY                   = var.container_number_map[var.instance_type].memory[local.get_compute_memory_index]
    "NVMES_NUM"                      = var.container_number_map[var.instance_type].nvme
    "TIERING_SSD_PERCENT"            = var.tiering_ssd_percent
    "PREFIX"                         = var.prefix
    "KEY_VAULT_URI"                  = azurerm_key_vault.key_vault.vault_uri
    "INSTALL_DPDK"                   = var.install_cluster_dpdk
    "NICS_NUM"                       = var.container_number_map[var.instance_type].nics
    "INSTALL_URL"                    = var.install_weka_url != "" ? var.install_weka_url : "https://$TOKEN@get.weka.io/dist/v1/install/${var.weka_version}/${var.weka_version}"
    "LOG_LEVEL"                      = var.function_app_log_level
    "SUBNET"                         = data.azurerm_subnet.subnet.address_prefix
    FUNCTION_APP_NAME                = local.function_app_name
    PROXY_URL                        = var.proxy_url

    https_only               = true
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
    ignore_changes = [site_config, tags]
  }

  depends_on = [azurerm_storage_account.deployment_sa, azurerm_subnet.subnet_delegation, module.network]
}

data "azurerm_subscription" "primary" {}

resource "azurerm_role_assignment" "storage-blob-data-contributor" {
  scope                = local.deployment_storage_account_scope
  role_definition_name = "Storage Blob Data Contributor"
  principal_id         = azurerm_linux_function_app.function_app.identity[0].principal_id
  depends_on           = [azurerm_linux_function_app.function_app, azurerm_storage_account.deployment_sa]
}

resource "azurerm_role_assignment" "storage_account_contributor" {
  scope                = data.azurerm_resource_group.rg.id
  role_definition_name = "Storage Account Contributor"
  principal_id         = azurerm_linux_function_app.function_app.identity[0].principal_id
  depends_on           = [azurerm_linux_function_app.function_app]
}

resource "azurerm_role_assignment" "obs_storage_blob_data_contributor" {
  count                = var.obs_name != "" ? 1 : 0
  scope                = local.obs_scope
  role_definition_name = "Storage Blob Data Contributor"
  principal_id         = azurerm_linux_function_app.function_app.identity[0].principal_id
  depends_on           = [azurerm_linux_function_app.function_app, azurerm_storage_account.deployment_sa]
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
  scope                = azurerm_linux_virtual_machine_scale_set.vmss.id
  role_definition_name = "Contributor"
  principal_id         = azurerm_linux_function_app.function_app.identity[0].principal_id
  depends_on           = [azurerm_linux_function_app.function_app, azurerm_linux_virtual_machine_scale_set.vmss]
}
