locals {
  create_private_function  = var.function_access_restriction_enabled ? 1 : 0
  stripe_width_calculated  = var.cluster_size - var.protection_level - 1
  stripe_width             = local.stripe_width_calculated < 16 ? local.stripe_width_calculated : 16
  location                 = data.azurerm_resource_group.rg.location
  read_remote_function_zip = !var.read_function_zip_from_storage_account
  function_app_zip_name    = local.read_remote_function_zip ? "${var.function_app_dist}/${var.function_app_version}.zip" : var.deployment_function_app_code_blob
  weka_sa                  = local.read_remote_function_zip ? "${var.function_app_storage_account_prefix}${local.location}" : var.deployment_storage_account_name
  weka_sa_container        = local.read_remote_function_zip ? "${var.function_app_storage_account_container_prefix}${local.location}" : var.deployment_container_name
  function_app_blob_sas    = local.read_remote_function_zip ? "" : data.azurerm_storage_account_blob_container_sas.function_app_code_sas[0].sas
  function_code_path       = "${path.module}/function-app/code"
  function_app_code_hash   = md5(join("", [for f in fileset(local.function_code_path, "**") : filemd5("${local.function_code_path}/${f}")]))
  get_compute_memory_index = var.set_dedicated_fe_container ? 1 : 0
  obs_storage_account_name = var.tiering_obs_name == "" ? "${substr("${local.alphanumeric_prefix_name}${local.alphanumeric_cluster_name}", 0, 21)}obs" : var.tiering_obs_name
  obs_container_name       = var.tiering_obs_container_name == "" ? "${var.prefix}-${var.cluster_name}-obs" : var.tiering_obs_container_name
  function_app_name        = "${local.alphanumeric_prefix_name}-${local.alphanumeric_cluster_name}-function-app"
  install_weka_url         = var.install_weka_url != "" ? var.install_weka_url : "https://$TOKEN@get.weka.io/dist/v1/install/${var.weka_version}/${var.weka_version}"
  supported_regions        = split("\n", replace(chomp(file("${path.module}/supported_regions/${var.function_app_dist}.txt")), "\r", ""))
  # log analytics for function app
  log_analytics_workspace_id   = var.enable_application_insights ? var.log_analytics_workspace_id == "" ? azurerm_log_analytics_workspace.la_workspace[0].id : var.log_analytics_workspace_id : ""
  application_insights_id      = var.enable_application_insights ? var.application_insights_name == "" ? azurerm_application_insights.application_insights[0].id : data.azurerm_application_insights.application_insights[0].id : ""
  application_insights_rg_name = var.application_insights_rg_name == "" ? var.rg_name : var.application_insights_rg_name
  insights_instrumenation_key  = var.enable_application_insights ? var.application_insights_name == "" ? azurerm_application_insights.application_insights[0].instrumentation_key : data.azurerm_application_insights.application_insights[0].instrumentation_key : null
  # nfs autoscaling
  nfs_deployment_container_name = var.nfs_deployment_container_name == "" ? "${local.alphanumeric_prefix_name}${local.alphanumeric_cluster_name}-protocol-deployment" : var.nfs_deployment_container_name

  clusterization_target = var.clusterization_target != null ? var.clusterization_target : min(var.cluster_size, max(20, ceil(var.cluster_size * 0.8)))
  # fields that depend on LB creation
  vmss_health_probe_id = var.create_lb ? azurerm_lb_probe.backend_lb_probe[0].id : null
  lb_backend_pool_ids  = var.create_lb ? [azurerm_lb_backend_address_pool.lb_backend_pool[0].id] : []

  vmss_config = jsonencode({
    name                            = "${var.prefix}-${var.cluster_name}-vmss"
    location                        = data.azurerm_resource_group.rg.location
    zones                           = var.zone != null ? [var.zone] : []
    resource_group_name             = var.rg_name
    sku                             = var.instance_type
    upgrade_mode                    = "Manual"
    health_probe_id                 = local.vmss_health_probe_id
    admin_username                  = var.vm_username
    ssh_public_key                  = local.public_ssh_key
    computer_name_prefix            = "${var.prefix}-${var.cluster_name}-backend"
    custom_data                     = ""
    disable_password_authentication = true
    proximity_placement_group_id    = local.placement_group_id
    single_placement_group          = var.vmss_single_placement_group
    source_image_id                 = var.source_image_id
    overprovision                   = false
    orchestration_mode              = "Uniform"
    tags = merge(var.tags_map, {
      "weka_cluster" : var.cluster_name,
      "user_id" : data.azurerm_client_config.current.object_id,
    })

    os_disk = {
      caching              = "ReadWrite"
      storage_account_type = "Premium_LRS"
    }

    data_disk = {
      lun                  = 0
      caching              = "None"
      create_option        = "Empty"
      disk_size_gb         = local.disk_size
      storage_account_type = "Premium_LRS"
    }

    identity = {
      type         = "UserAssigned"
      identity_ids = [local.vmss_identity_id]
    }

    primary_nic = {
      name                          = "${var.prefix}-${var.cluster_name}-backend-nic-0"
      network_security_group_id     = local.sg_id
      enable_accelerated_networking = var.install_cluster_dpdk

      ip_configurations = [{
        primary                                = true
        subnet_id                              = data.azurerm_subnet.subnet.id
        load_balancer_backend_address_pool_ids = local.lb_backend_pool_ids
        public_ip_address = {
          assign            = local.assign_public_ip
          name              = "${var.prefix}-${var.cluster_name}-public-ip"
          domain_name_label = "${var.prefix}-${var.cluster_name}-backend"
        }
      }]
    }

    secondary_nics = {
      number                        = local.nics_numbers - 1
      name_prefix                   = "${var.prefix}-${var.cluster_name}-backend-nic"
      network_security_group_id     = local.sg_id
      enable_accelerated_networking = var.install_cluster_dpdk
      ip_configurations = [{
        primary                                = true
        subnet_id                              = data.azurerm_subnet.subnet.id
        load_balancer_backend_address_pool_ids = local.lb_backend_pool_ids
      }]
    }
  })

  # function app settings
  initial_app_settings = {
    "USER_ASSIGNED_CLIENT_ID"      = local.function_app_identity_client_id
    "STATE_STORAGE_NAME"           = local.deployment_storage_account_name
    "STATE_CONTAINER_NAME"         = local.deployment_container_name
    "STATE_BLOB_NAME"              = "state"
    "HOSTS_NUM"                    = var.cluster_size
    "CLUSTER_NAME"                 = var.cluster_name
    "PROTECTION_LEVEL"             = var.protection_level
    "STRIPE_WIDTH"                 = var.stripe_width != -1 ? var.stripe_width : local.stripe_width
    "HOTSPARE"                     = var.hotspare
    "VM_USERNAME"                  = var.vm_username
    "SUBSCRIPTION_ID"              = var.subscription_id
    "RESOURCE_GROUP_NAME"          = data.azurerm_resource_group.rg.name
    "LOCATION"                     = data.azurerm_resource_group.rg.location
    "SET_OBS"                      = var.tiering_enable_obs_integration
    "CREATE_CONFIG_FS"             = (var.smbw_enabled && var.smb_setup_protocol) || var.s3_setup_protocol
    "OBS_NAME"                     = local.obs_storage_account_name
    "OBS_CONTAINER_NAME"           = local.obs_container_name
    "OBS_ACCESS_KEY"               = var.tiering_blob_obs_access_key
    "OBS_NETWORK_ACCESS"           = var.storage_account_public_network_access
    "OBS_ALLOWED_SUBNETS"          = join(",", local.sa_public_access_for_vnet ? [data.azurerm_subnet.subnet.id, local.function_app_subnet_delegation_id] : [])
    "OBS_ALLOWED_PUBLIC_IPS"       = join(",", var.storage_account_allowed_ips)
    DRIVE_CONTAINER_CORES_NUM      = var.containers_config_map[var.instance_type].drive
    COMPUTE_CONTAINER_CORES_NUM    = var.set_dedicated_fe_container == false ? var.containers_config_map[var.instance_type].compute + 1 : var.containers_config_map[var.instance_type].compute
    FRONTEND_CONTAINER_CORES_NUM   = var.set_dedicated_fe_container == false ? 0 : var.containers_config_map[var.instance_type].frontend
    COMPUTE_MEMORY                 = var.containers_config_map[var.instance_type].memory[local.get_compute_memory_index]
    DISK_SIZE                      = local.disk_size
    "NVMES_NUM"                    = var.containers_config_map[var.instance_type].nvme
    "TIERING_SSD_PERCENT"          = var.tiering_enable_ssd_percent
    "TIERING_TARGET_SSD_RETENTION" = var.tiering_obs_target_ssd_retention
    "TIERING_START_DEMOTE"         = var.tiering_obs_start_demote
    "PREFIX"                       = var.prefix
    "KEY_VAULT_URI"                = azurerm_key_vault.key_vault.vault_uri
    "INSTALL_DPDK"                 = var.install_cluster_dpdk
    "NICS_NUM"                     = var.containers_config_map[var.instance_type].nics
    "INSTALL_URL"                  = local.install_weka_url
    "LOG_LEVEL"                    = var.function_app_log_level
    "SUBNET"                       = local.subnet_range
    "SUBNET_ID"                    = data.azurerm_subnet.subnet.id
    "BLOB_PRIVATE_DNS_ZONE_ID"     = var.create_storage_account_private_links ? azurerm_private_dns_zone.blob[0].id : local.sa_public_access_disabled ? data.azurerm_private_dns_zone.blob[0].id : ""
    "CREATE_BLOB_PRIVATE_ENDPOINT" = var.create_storage_account_private_links && local.sa_public_access_disabled
    FUNCTION_APP_NAME              = local.function_app_name
    PROXY_URL                      = var.proxy_url
    WEKA_HOME_URL                  = var.weka_home_url
    POST_CLUSTER_CREATION_SCRIPT   = var.script_post_cluster_creation
    PRE_START_IO_SCRIPT            = var.script_pre_start_io
    DOWN_BACKENDS_REMOVAL_TIMEOUT  = var.debug_down_backends_removal_timeout

    https_only               = true
    FUNCTION_APP_EDIT_MODE   = "readonly"
    HASH                     = var.function_app_version
    WEBSITE_RUN_FROM_PACKAGE = "https://${local.weka_sa}.blob.core.windows.net/${local.weka_sa_container}/${local.function_app_zip_name}${local.function_app_blob_sas}"

    NFS_STATE_CONTAINER_NAME          = local.nfs_deployment_container_name
    NFS_STATE_BLOB_NAME               = "nfs_state"
    NFS_INTERFACE_GROUP_NAME          = var.nfs_interface_group_name
    NFS_SECONDARY_IPS_NUM             = var.nfs_protocol_gateway_secondary_ips_per_nic
    NFS_PROTOCOL_GATEWAY_FE_CORES_NUM = var.nfs_protocol_gateway_fe_cores_num
    NFS_PROTOCOL_GATEWAYS_NUM         = var.nfs_protocol_gateways_number
    NFS_VMSS_NAME                     = var.nfs_protocol_gateways_number > 0 ? "${var.prefix}-${var.cluster_name}-nfs-protocol-gateway-vmss" : ""
    NFS_DISK_SIZE                     = var.nfs_protocol_gateway_disk_size
    SMB_DISK_SIZE                     = var.smb_protocol_gateway_disk_size
    S3_DISK_SIZE                      = var.s3_protocol_gateway_disk_size
    SMB_PROTOCOL_GATEWAY_FE_CORES_NUM = var.smb_protocol_gateway_fe_cores_num
    S3_PROTOCOL_GATEWAY_FE_CORES_NUM  = var.s3_protocol_gateway_fe_cores_num
    TRACES_PER_FRONTEND               = var.traces_per_ionode
    SET_DEFAULT_FS                    = var.set_default_fs
    POST_CLUSTER_SETUP_SCRIPT         = var.post_cluster_setup_script

    BACKEND_LB_IP = var.create_lb ? azurerm_lb.backend_lb[0].private_ip_address : ""
    # state
    INITIAL_CLUSTER_SIZE  = var.cluster_size
    CLUSTERIZATION_TARGET = local.clusterization_target
    VMSS_CONFIG           = local.vmss_config
    # init script inputs
    APT_REPO_SERVER = var.apt_repo_server
    USER_DATA       = var.user_data
  }

  secured_storage_account_app_settings = {
    # "AzureWebJobsStorage" and "WEBSITE_CONTENTAZUREFILECONNECTIONSTRING" are not needed as we set storage_account_access_key
    "WEBSITE_CONTENTSHARE"    = local.deployment_file_share_name
    "WEBSITE_CONTENTOVERVNET" = 1
    "WEBSITE_VNET_ROUTE_ALL"  = 1
  }
  dns_storgage_account_app_settings = {
    "WEBSITE_DNS_SERVER" = "168.63.129.16"
  }
  merged_secured_storage_account_app_settings = var.create_storage_account_private_links ? merge(local.secured_storage_account_app_settings, local.dns_storgage_account_app_settings) : local.secured_storage_account_app_settings

  app_settings = local.sa_public_access_enabled ? local.initial_app_settings : merge(local.initial_app_settings, local.merged_secured_storage_account_app_settings)

  function_app_subnet_delegation_id = var.function_app_subnet_delegation_id == "" ? module.function_app_subnet_delegation[0].id : var.function_app_subnet_delegation_id
}

resource "azurerm_log_analytics_workspace" "la_workspace" {
  count               = var.log_analytics_workspace_id == "" && var.enable_application_insights ? 1 : 0
  name                = "${local.alphanumeric_prefix_name}-${local.alphanumeric_cluster_name}-workspace"
  location            = data.azurerm_resource_group.rg.location
  resource_group_name = var.rg_name
  sku                 = "PerGB2018"
  retention_in_days   = 30
  lifecycle {
    ignore_changes = [tags]
  }
}

data "azurerm_application_insights" "application_insights" {
  count               = var.application_insights_name != "" && var.enable_application_insights ? 1 : 0
  name                = var.application_insights_name
  resource_group_name = local.application_insights_rg_name
}

data "azurerm_resource_group" "application_insights_rg" {
  count = var.application_insights_name == "" && var.enable_application_insights ? 1 : 0
  name  = local.application_insights_rg_name
}

resource "azurerm_application_insights" "application_insights" {
  count               = var.application_insights_name == "" && var.enable_application_insights ? 1 : 0
  name                = "${var.prefix}-${var.cluster_name}-application-insights"
  location            = data.azurerm_resource_group.application_insights_rg[0].location
  resource_group_name = local.application_insights_rg_name
  workspace_id        = local.log_analytics_workspace_id
  application_type    = "web"
  lifecycle {
    ignore_changes = [tags]
  }
}

resource "azurerm_monitor_diagnostic_setting" "insights_diagnostic_setting" {
  count                      = var.enable_application_insights ? 1 : 0
  name                       = "${var.prefix}-${var.cluster_name}-insights-diagnostic-setting"
  target_resource_id         = local.application_insights_id
  storage_account_id         = local.deployment_storage_account_id
  log_analytics_workspace_id = local.log_analytics_workspace_id
  enabled_log {
    category = "AppTraces"
  }
  lifecycle {
    ignore_changes = [metric, log_analytics_destination_type]
  }
  depends_on = [azurerm_linux_function_app.function_app]
}

resource "azurerm_monitor_diagnostic_setting" "function_diagnostic_setting" {
  count                      = var.enable_application_insights ? 1 : 0
  name                       = "${var.prefix}-${var.cluster_name}-function-diagnostic-setting"
  target_resource_id         = azurerm_linux_function_app.function_app.id
  storage_account_id         = local.deployment_storage_account_id
  log_analytics_workspace_id = local.log_analytics_workspace_id
  enabled_log {
    category = "FunctionAppLogs"
  }
  lifecycle {
    ignore_changes = [metric, log_analytics_destination_type]
  }
  depends_on = [azurerm_linux_function_app.function_app]
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
  storage_account_access_key = local.deployment_sa_access_key
  https_only                 = true
  virtual_network_subnet_id  = local.function_app_subnet_delegation_id

  site_config {
    vnet_route_all_enabled   = true
    application_insights_key = local.insights_instrumenation_key
    application_stack {
      use_custom_runtime = true
    }
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
      for_each = range(local.sa_public_access_enabled ? local.create_private_function : 0)
      content {
        virtual_network_subnet_id = var.logic_app_subnet_delegation_id == "" ? module.logic_app_subnet_delegation[0].id : var.logic_app_subnet_delegation_id
        action                    = "Allow"
        priority                  = 301
        name                      = "VirtualNetwork"
      }
    }
  }

  app_settings = local.app_settings

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

    precondition {
      condition     = local.sa_public_access_enabled || local.sa_public_access_for_vnet && local.sa_allowed_ips_provided || (local.sa_public_access_disabled || local.sa_public_access_for_vnet && !local.sa_allowed_ips_provided) && var.deployment_storage_account_name != ""
      error_message = "You shoud pick one of 3 options: 1. Public access enabled, 2. Public access enabled for VNET + public IPs whitelisted, 3. Public access disabled (or enabled for VNET without IPs whitelisted) and deployment_storage_account_name provided"
    }

    precondition {
      condition     = local.read_remote_function_zip || var.deployment_function_app_code_blob != ""
      error_message = "You should provide value for 'deployment_function_app_code_blob' or 'read_function_zip_from_storage_account' should be false"
    }

    precondition {
      condition     = var.install_weka_url != "" || var.weka_version != ""
      error_message = "Please provide either 'install_weka_url' or 'weka_version' variables."
    }
  }

  depends_on = [module.network, module.iam, azurerm_storage_account.deployment_sa, azurerm_private_endpoint.file_endpoint, azurerm_private_endpoint.blob_endpoint]
}
