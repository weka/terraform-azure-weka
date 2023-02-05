resource "azurerm_log_analytics_workspace" "la_workspace" {
  name                = "${var.prefix}-${var.cluster_name}-workspace"
  location            = data.azurerm_resource_group.rg.location
  resource_group_name = data.azurerm_resource_group.rg.name
  sku                 = "PerGB2018"
  retention_in_days   = 30
}

resource "azurerm_application_insights" "application_insights" {
  name = "${var.prefix}-${var.cluster_name}-application-insights"
  location= data.azurerm_resource_group.rg.location
  resource_group_name = data.azurerm_resource_group.rg.name
  workspace_id        = azurerm_log_analytics_workspace.la_workspace.id
  application_type    = "web"
}

resource "azurerm_monitor_diagnostic_setting" "diagnostic_setting" {
  name                       = "${var.prefix}-${var.cluster_name}-diagnostic-setting"
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
  function_zip_path = "/tmp/${var.prefix}-${var.cluster_name}-function-app.zip"
  function_code_path = "${path.module}/function-app/"
}

resource "null_resource" "build_function_code" {
  triggers = {
    always_run = timestamp()
  }

  provisioner "local-exec" {
    command = <<EOT
    cd ${path.module}/function-app
    go mod tidy
    GOOS=linux GOARCH=amd64 go build
    EOT
  }
}

data "archive_file" "function_zip" {
  type        = "zip"
  output_path = local.function_zip_path
  source_dir  = local.function_code_path
  depends_on = [null_resource.build_function_code]
}

resource "azurerm_storage_blob" "function_app_code" {
  name = "function_app.zip"
  storage_account_name = azurerm_storage_account.deployment_sa.name
  storage_container_name = azurerm_storage_container.deployment.name
  type = "Block"
  source = data.archive_file.function_zip.output_path
  content_md5 = data.archive_file.function_zip.output_md5
  depends_on = [data.archive_file.function_zip]
}


locals {
  stripe_width_calculated = var.cluster_size - var.protection_level - 1
  stripe_width = local.stripe_width_calculated < 16 ? local.stripe_width_calculated : 16
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
    "TIERING_SSD_PERCENT" = var.tiering_ssd_percent
    "PREFIX" = var.prefix
    "KEY_VAULT_URI" = azurerm_key_vault.key_vault.vault_uri
    "SUBNET" = var.subnets[0]
    "INSTANCE_TYPE" = var.instance_type
    "INSTALL_URL" =  var.install_weka_url != "" ? var.install_weka_url : "https://$TOKEN@get.weka.io/dist/v1/install/${var.weka_version}/${var.weka_version}"
    "LOG_LEVEL" = var.function_app_log_level

    https_only = true
    FUNCTIONS_WORKER_RUNTIME = "custom"
    FUNCTION_APP_EDIT_MODE   = "readonly"
    HASH                     = azurerm_storage_blob.function_app_code.content_md5
    WEBSITE_RUN_FROM_PACKAGE = "https://${azurerm_storage_account.deployment_sa.name}.blob.core.windows.net/${azurerm_storage_container.deployment.name}/${azurerm_storage_blob.function_app_code.name}"
    WEBSITE_VNET_ROUTE_ALL   = true
  }

  identity {
    type = "SystemAssigned"
  }

  depends_on = [azurerm_storage_account.deployment_sa, azurerm_storage_blob.function_app_code]
}

# service principal

data "azuread_client_config" "function-app-client-config" {}

data "azurerm_subscription" "primary" {}

resource "azuread_application" "function_app" {
  display_name = "function-app"
  owners       = [data.azuread_client_config.function-app-client-config.object_id]
}

resource "azuread_service_principal" "function-app-principal" {
  application_id               = azuread_application.function_app.application_id
  app_role_assignment_required = false
  owners                       = [data.azuread_client_config.function-app-client-config.object_id]
}

resource "azurerm_role_assignment" "function-assignment" {
  scope                = data.azurerm_subscription.primary.id
  role_definition_name = "Storage Blob Data Contributor"
  principal_id         = azuread_service_principal.function-app-principal.id
  depends_on           = [azuread_service_principal.function-app-principal, azuread_application.function_app]
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
  scope                = azurerm_linux_virtual_machine_scale_set.vmss.id
  role_definition_name = "Owner"
  principal_id         = azurerm_linux_function_app.function_app.identity[0].principal_id
  depends_on           = [azurerm_linux_function_app.function_app]
}
