locals {
  create_private_function         = var.function_access_restriction_enabled ? 1 : 0
  stripe_width_calculated         = var.cluster_size - var.protection_level - 1
  stripe_width                    = local.stripe_width_calculated < 16 ? local.stripe_width_calculated : 16
  location                        = data.azurerm_resource_group.rg.location
  function_app_zip_name           = "${var.function_app_dist}/${var.function_app_version}.zip"
  weka_sa                         = "${var.function_app_storage_account_prefix}eastus"
  weka_sa_container               = "${var.function_app_storage_account_container_prefix}eastus"
  function_code_path              = "${path.module}/function-app/code"
  function_app_code_hash          = md5(join("", [for f in fileset(local.function_code_path, "**") : filemd5("${local.function_code_path}/${f}")]))
  get_compute_memory_index        = var.set_dedicated_fe_container ? 1 : 0
  deployment_storage_account_id   = var.deployment_storage_account_name == "" ? azurerm_storage_account.deployment_sa[0].id : data.azurerm_storage_account.deployment_blob[0].id
  deployment_storage_account_name = var.deployment_storage_account_name == "" ? azurerm_storage_account.deployment_sa[0].name : var.deployment_storage_account_name
  deployment_container_name       = var.deployment_container_name == "" ? azurerm_storage_container.deployment[0].name : var.deployment_container_name
  obs_storage_account_name        = var.tiering_obs_name == "" ? "${substr("${local.alphanumeric_prefix_name}${local.alphanumeric_cluster_name}", 0, 21)}obs" : var.tiering_obs_name
  obs_container_name              = var.tiering_obs_container_name == "" ? "${var.prefix}-${var.cluster_name}-obs" : var.tiering_obs_container_name
  function_app_name               = "${local.alphanumeric_prefix_name}-${local.alphanumeric_cluster_name}-function-app"
  install_weka_url                = var.install_weka_url != "" ? var.install_weka_url : "https://$TOKEN@get.weka.io/dist/v1/install/${var.weka_version}/${var.weka_version}"
  supported_regions               = split("\n", replace(chomp(file("${path.module}/supported_regions/${var.function_app_dist}.txt")), "\r", ""))
  # log analytics for function app
  #log_analytics_workspace_id  = var.enable_application_insights ? var.log_analytics_workspace_id == "" ? azurerm_log_analytics_workspace.la_workspace[0].id : var.log_analytics_workspace_id : ""
  #application_insights_id     = var.enable_application_insights ? var.application_insights_name == "" ? azurerm_application_insights.application_insights[0].id : data.azurerm_application_insights.application_insights[0].id : ""
  #insights_instrumenation_key = var.enable_application_insights ? var.application_insights_name == "" ? azurerm_application_insights.application_insights[0].instrumentation_key : data.azurerm_application_insights.application_insights[0].instrumentation_key : ""
}

# resource "azurerm_log_analytics_workspace" "la_workspace" {
#   count               = var.log_analytics_workspace_id == "" && var.enable_application_insights ? 1 : 0
#   name                = "${local.alphanumeric_prefix_name}-${local.alphanumeric_cluster_name}-workspace"
#   location            = data.azurerm_resource_group.rg.location
#   resource_group_name = data.azurerm_resource_group.rg.name
#   sku                 = "PerGB2018"
#   retention_in_days   = 30
#   lifecycle {
#     ignore_changes = [tags]
#   }
# }

# data "azurerm_application_insights" "application_insights" {
#   count               = var.application_insights_name != "" && var.enable_application_insights ? 1 : 0
#   name                = var.application_insights_name
#   resource_group_name = data.azurerm_resource_group.rg.name
# }

# resource "azurerm_application_insights" "application_insights" {
#   count               = var.application_insights_name == "" && var.enable_application_insights ? 1 : 0
#   name                = "${var.prefix}-${var.cluster_name}-application-insights"
#   location            = data.azurerm_resource_group.rg.location
#   resource_group_name = data.azurerm_resource_group.rg.name
#   workspace_id        = local.log_analytics_workspace_id
#   application_type    = "web"
#   lifecycle {
#     ignore_changes = [tags]
#   }
# }

# resource "azurerm_monitor_diagnostic_setting" "insights_diagnostic_setting" {
#   count                      = var.enable_application_insights ? 1 : 0
#   name                       = "${var.prefix}-${var.cluster_name}-insights-diagnostic-setting"
#   target_resource_id         = local.application_insights_id
#   storage_account_id         = local.deployment_storage_account_id
#   log_analytics_workspace_id = local.log_analytics_workspace_id
#   enabled_log {
#     category = "AppTraces"
#   }
#   lifecycle {
#     ignore_changes = [metric, log_analytics_destination_type]
#   }
#   depends_on = [azurerm_linux_function_app.function_app]
# }

# resource "azurerm_monitor_diagnostic_setting" "function_diagnostic_setting" {
#   count                      = var.enable_application_insights ? 1 : 0
#   name                       = "${var.prefix}-${var.cluster_name}-function-diagnostic-setting"
#   target_resource_id         = azurerm_linux_function_app.function_app.id
#   storage_account_id         = local.deployment_storage_account_id
#   log_analytics_workspace_id = local.log_analytics_workspace_id
#   enabled_log {
#     category = "FunctionAppLogs"
#   }
#   lifecycle {
#     ignore_changes = [metric, log_analytics_destination_type]
#   }
#   depends_on = [azurerm_linux_function_app.function_app]
# }

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

resource "azurerm_subnet" "subnet_delegation" {
  count                = var.function_app_subnet_delegation_id == "" ? 1 : 0
  name                 = "${var.prefix}-${var.cluster_name}-subnet-delegation"
  resource_group_name  = local.vnet_rg_name
  virtual_network_name = local.vnet_name
  address_prefixes     = [var.function_app_subnet_delegation_cidr]
  service_endpoints    = ["Microsoft.Storage", "Microsoft.KeyVault", "Microsoft.Web"]
  delegation {
    name = "subnet-delegation"
    service_delegation {
      name    = "Microsoft.Web/serverFarms"
      actions = ["Microsoft.Network/virtualNetworks/subnets/action"]
    }
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
  virtual_network_subnet_id  = var.function_app_subnet_delegation_id == "" ? azurerm_subnet.subnet_delegation[0].id : var.function_app_subnet_delegation_id
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
    dynamic "ip_restriction" {
      for_each = range(local.create_private_function)
      content {
        virtual_network_subnet_id = var.logic_app_subnet_delegation_id == "" ? azurerm_subnet.logicapp_subnet_delegation[0].id : var.logic_app_subnet_delegation_id
        action                    = "Allow"
        priority                  = 301
        name                      = "VirtualNetwork"
      }
    }
  }

  app_settings = {
    "USER_ASSIGNED_CLIENT_ID"        = local.function_app_identity_client_id
    #"APPINSIGHTS_INSTRUMENTATIONKEY" = local.insights_instrumenation_key
    "STATE_STORAGE_NAME"             = local.deployment_storage_account_name
    "STATE_CONTAINER_NAME"           = local.deployment_container_name
    "HOSTS_NUM"                      = var.cluster_size
    "CLUSTER_NAME"                   = var.cluster_name
    "PROTECTION_LEVEL"               = var.protection_level
    "STRIPE_WIDTH"                   = var.stripe_width != -1 ? var.stripe_width : local.stripe_width
    "HOTSPARE"                       = var.hotspare
    "VM_USERNAME"                    = var.vm_username
    "SUBSCRIPTION_ID"                = var.subscription_id
    "RESOURCE_GROUP_NAME"            = data.azurerm_resource_group.rg.name
    "LOCATION"                       = data.azurerm_resource_group.rg.location
    "SET_OBS"                        = var.tiering_enable_obs_integration
    "SMBW_ENABLED"                   = var.smbw_enabled
    "OBS_NAME"                       = local.obs_storage_account_name
    "OBS_CONTAINER_NAME"             = local.obs_container_name
    "OBS_ACCESS_KEY"                 = var.tiering_blob_obs_access_key
    DRIVE_CONTAINER_CORES_NUM        = var.containers_config_map[var.instance_type].drive
    COMPUTE_CONTAINER_CORES_NUM      = var.set_dedicated_fe_container == false ? var.containers_config_map[var.instance_type].compute + 1 : var.containers_config_map[var.instance_type].compute
    FRONTEND_CONTAINER_CORES_NUM     = var.set_dedicated_fe_container == false ? 0 : var.containers_config_map[var.instance_type].frontend
    COMPUTE_MEMORY                   = var.containers_config_map[var.instance_type].memory[local.get_compute_memory_index]
    "NVMES_NUM"                      = var.containers_config_map[var.instance_type].nvme
    "TIERING_SSD_PERCENT"            = var.tiering_enable_ssd_percent
    "PREFIX"                         = var.prefix
    "KEY_VAULT_URI"                  = azurerm_key_vault.key_vault.vault_uri
    "INSTALL_DPDK"                   = var.install_cluster_dpdk
    "NICS_NUM"                       = var.containers_config_map[var.instance_type].nics
    "INSTALL_URL"                    = local.install_weka_url
    "LOG_LEVEL"                      = var.function_app_log_level
    "SUBNET"                         = data.azurerm_subnet.subnet.address_prefix
    FUNCTION_APP_NAME                = local.function_app_name
    PROXY_URL                        = var.proxy_url
    WEKA_HOME_URL                    = var.weka_home_url
    POST_CLUSTER_CREATION_SCRIPT     = var.script_post_cluster_creation
    PRE_START_IO_SCRIPT              = var.script_pre_start_io

    https_only                  = true
    FUNCTIONS_EXTENSION_VERSION = "~4"
    FUNCTIONS_WORKER_RUNTIME    = "custom"
    FUNCTION_APP_EDIT_MODE      = "readonly"
    HASH                        = var.function_app_version
    WEBSITE_RUN_FROM_PACKAGE    = "https://${local.weka_sa}.blob.core.windows.net/${local.weka_sa_container}/${local.function_app_zip_name}"
    WEBSITE_VNET_ROUTE_ALL      = true
  }

  identity {
    type         = "UserAssigned"
    identity_ids = [local.function_app_identity_id]
  }

  lifecycle {
    precondition {
      condition     = var.function_app_dist == "release" || var.function_app_version == local.function_app_code_hash
      error_message = "Please update function app code version."
    }
    ignore_changes = [site_config, tags]

    precondition {
      condition     = contains(local.supported_regions, data.azurerm_resource_group.rg.location)
      error_message = "The region '${data.azurerm_resource_group.rg.location}' is not supported for the function_app_dist '${var.function_app_dist}'. Supported regions: ${join(", ", local.supported_regions)}"
    }
  }

  depends_on = [module.network, azurerm_storage_account.deployment_sa, azurerm_subnet.logicapp_subnet_delegation]
}
