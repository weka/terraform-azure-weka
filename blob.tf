locals {
  deployment_storage_account_id   = var.deployment_storage_account_name == "" ? azurerm_storage_account.deployment_sa[0].id : data.azurerm_storage_account.deployment_blob[0].id
  deployment_storage_account_name = var.deployment_storage_account_name == "" ? azurerm_storage_account.deployment_sa[0].name : var.deployment_storage_account_name
  deployment_sa_connection_string = var.deployment_storage_account_name == "" ? azurerm_storage_account.deployment_sa[0].primary_connection_string : data.azurerm_storage_account.deployment_blob[0].primary_connection_string
  deployment_container_name       = var.deployment_container_name == "" ? "${local.alphanumeric_prefix_name}${local.alphanumeric_cluster_name}-deployment" : var.deployment_container_name
  deployment_sa_access_key        = var.deployment_storage_account_name == "" ? azurerm_storage_account.deployment_sa[0].primary_access_key : data.azurerm_storage_account.deployment_blob[0].primary_access_key

  sa_allowed_ips_provided   = length(var.storage_account_allowed_ips) > 0
  sa_public_access_enabled  = var.storage_account_public_network_access == "Enabled"
  sa_public_access_for_vnet = var.storage_account_public_network_access == "EnabledForVnet"
  sa_public_access_disabled = var.storage_account_public_network_access == "Disabled"
  create_sa_resources       = local.sa_public_access_enabled || local.sa_public_access_for_vnet && local.sa_allowed_ips_provided
}

resource "azurerm_storage_account" "deployment_sa" {
  count                    = var.deployment_storage_account_name == "" && local.create_sa_resources ? 1 : 0
  name                     = substr("${local.alphanumeric_prefix_name}${local.alphanumeric_cluster_name}deployment", 0, 24)
  location                 = local.location
  resource_group_name      = var.rg_name
  account_kind             = "StorageV2"
  account_tier             = "Standard"
  account_replication_type = "ZRS"
  tags                     = merge(var.tags_map, { "weka_cluster" : var.cluster_name })
  lifecycle {
    ignore_changes = [tags]
  }

  dynamic "network_rules" {
    for_each = local.sa_public_access_for_vnet ? [1] : []
    content {
      default_action             = "Deny"
      bypass                     = ["AzureServices"]
      ip_rules                   = var.storage_account_allowed_ips
      virtual_network_subnet_ids = [data.azurerm_subnet.subnet.id, local.function_app_subnet_delegation_id]
    }
  }
}

resource "azurerm_storage_container" "deployment" {
  count                 = var.deployment_container_name == "" && local.create_sa_resources ? 1 : 0
  name                  = "${local.alphanumeric_prefix_name}${local.alphanumeric_cluster_name}-deployment"
  storage_account_name  = local.deployment_storage_account_name
  container_access_type = "private"
  depends_on            = [azurerm_storage_account.deployment_sa]
}

resource "azurerm_storage_blob" "state" {
  count                  = local.create_sa_resources ? 1 : 0
  name                   = "state"
  storage_account_name   = local.deployment_storage_account_name
  storage_container_name = local.deployment_container_name
  type                   = "Block"
  source_content         = "{\"initial_size\":${var.cluster_size}, \"desired_size\":${var.cluster_size}, \"instances\":[], \"clusterized\":false, \"clusterization_target\":${local.clusterization_target}}"
  depends_on             = [azurerm_storage_container.deployment]

  lifecycle {
    ignore_changes = all
  }
}

resource "azurerm_storage_share" "function_app_share" {
  count                = local.sa_public_access_for_vnet && local.sa_allowed_ips_provided ? 1 : 0
  name                 = "${local.deployment_container_name}-share"
  storage_account_name = local.deployment_storage_account_name
  quota                = 100
  depends_on           = [azurerm_storage_account.deployment_sa]
}


# state for protocols
resource "azurerm_storage_container" "nfs_deployment" {
  count                 = var.nfs_deployment_container_name == "" && local.create_sa_resources ? 1 : 0
  name                  = "${local.alphanumeric_prefix_name}${local.alphanumeric_cluster_name}-protocol-deployment"
  storage_account_name  = local.deployment_storage_account_name
  container_access_type = "private"
  depends_on            = [azurerm_storage_account.deployment_sa]
}

resource "azurerm_storage_blob" "nfs_state" {
  count                  = var.nfs_protocol_gateways_number > 0 && local.create_sa_resources ? 1 : 0
  name                   = "nfs_state"
  storage_account_name   = local.deployment_storage_account_name
  storage_container_name = local.nfs_deployment_container_name
  type                   = "Block"
  source_content = jsonencode({
    initial_size          = var.nfs_protocol_gateways_number
    desired_size          = var.nfs_protocol_gateways_number
    instances             = []
    clusterized           = false
    clusterization_target = var.nfs_protocol_gateways_number
  })
  depends_on = [azurerm_storage_container.nfs_deployment]

  lifecycle {
    ignore_changes = all
  }
}

resource "azurerm_storage_account" "logicapp" {
  count                    = local.create_sa_resources ? 1 : 0
  name                     = substr("${local.alphanumeric_prefix_name}${local.alphanumeric_cluster_name}logicappsa", 0, 24)
  resource_group_name      = var.rg_name
  location                 = local.location
  account_tier             = "Standard"
  account_replication_type = "LRS"

  dynamic "network_rules" {
    for_each = local.sa_public_access_for_vnet ? [1] : []
    content {
      default_action             = "Deny"
      bypass                     = ["AzureServices"]
      ip_rules                   = var.storage_account_allowed_ips
      virtual_network_subnet_ids = [data.azurerm_subnet.subnet.id, var.logic_app_subnet_delegation_id == "" ? module.logic_app_subnet_delegation[0].id : var.logic_app_subnet_delegation_id]
    }
  }
}

data "azurerm_storage_account" "deployment_blob" {
  count               = var.deployment_storage_account_name != "" ? 1 : 0
  name                = var.deployment_storage_account_name
  resource_group_name = local.resource_group_name
}

resource "azurerm_private_dns_zone" "blob" {
  count               = var.create_storage_account_private_links ? 1 : 0
  name                = "privatelink.blob.core.windows.net"
  resource_group_name = local.resource_group_name
}

resource "azurerm_private_dns_zone" "file" {
  count               = var.create_storage_account_private_links ? 1 : 0
  name                = "privatelink.file.core.windows.net"
  resource_group_name = local.resource_group_name
}

resource "azurerm_private_dns_zone_virtual_network_link" "blob_privatelink" {
  count                 = var.create_storage_account_private_links ? 1 : 0
  name                  = "${var.prefix}-${var.cluster_name}-blob-privatelink"
  resource_group_name   = local.resource_group_name
  private_dns_zone_name = azurerm_private_dns_zone.blob[0].name
  virtual_network_id    = data.azurerm_virtual_network.vnet.id
}

resource "azurerm_private_dns_zone_virtual_network_link" "file_privatelink" {
  count                 = var.create_storage_account_private_links ? 1 : 0
  name                  = "${var.prefix}-${var.cluster_name}-file-privatelink"
  resource_group_name   = local.resource_group_name
  private_dns_zone_name = azurerm_private_dns_zone.file[0].name
  virtual_network_id    = data.azurerm_virtual_network.vnet.id
}

resource "azurerm_private_endpoint" "file_endpoint" {
  count               = var.create_storage_account_private_links ? 1 : 0
  name                = "${var.prefix}-${var.cluster_name}-file-endpoint"
  location            = data.azurerm_resource_group.rg.location
  resource_group_name = data.azurerm_resource_group.rg.name
  subnet_id           = data.azurerm_subnet.subnet.id
  tags                = merge(var.tags_map, { "weka_cluster" : var.cluster_name })

  private_dns_zone_group {
    name                 = "${var.prefix}-${var.cluster_name}-dns-zone-group-file"
    private_dns_zone_ids = [azurerm_private_dns_zone.file[0].id]
  }

  private_service_connection {
    name                           = "${var.prefix}-${var.cluster_name}-privateFileSvcCon"
    is_manual_connection           = false
    private_connection_resource_id = local.deployment_storage_account_id
    subresource_names              = ["file"]
  }
}

resource "azurerm_private_endpoint" "blob_endpoint" {
  count               = var.create_storage_account_private_links ? 1 : 0
  name                = "${var.prefix}-${var.cluster_name}-blob-endpoint"
  location            = data.azurerm_resource_group.rg.location
  resource_group_name = data.azurerm_resource_group.rg.name
  subnet_id           = data.azurerm_subnet.subnet.id
  tags                = merge(var.tags_map, { "weka_cluster" : var.cluster_name })

  private_dns_zone_group {
    name                 = "${var.prefix}-${var.cluster_name}-dns-zone-group-blob"
    private_dns_zone_ids = [azurerm_private_dns_zone.blob[0].id]
  }
  private_service_connection {
    name                           = "${var.prefix}-${var.cluster_name}-privateBlobSvcCon"
    is_manual_connection           = false
    private_connection_resource_id = local.deployment_storage_account_id
    subresource_names              = ["blob"]
  }
}

data "azurerm_storage_account" "weka_obs" {
  count               = var.tiering_obs_name != "" ? 1 : 0
  name                = var.tiering_obs_name
  resource_group_name = var.rg_name
}

resource "azurerm_private_endpoint" "weka_obs_blob_endpoint" {
  count               = var.create_storage_account_private_links && var.tiering_blob_obs_access_key != "" ? 1 : 0
  name                = "${var.prefix}-${var.cluster_name}-obs-blob-endpoint"
  location            = data.azurerm_resource_group.rg.location
  resource_group_name = data.azurerm_resource_group.rg.name
  subnet_id           = data.azurerm_subnet.subnet.id
  tags                = merge(var.tags_map, { "weka_cluster" : var.cluster_name })

  private_dns_zone_group {
    name                 = "${var.prefix}-${var.cluster_name}-dns-zone-group-obs-blob"
    private_dns_zone_ids = [azurerm_private_dns_zone.blob[0].id]
  }
  private_service_connection {
    name                           = "${var.prefix}-${var.cluster_name}-private-obs-BlobSvcCon"
    is_manual_connection           = false
    private_connection_resource_id = data.azurerm_storage_account.weka_obs[0].id
    subresource_names              = ["blob"]
  }

  lifecycle {
    precondition {
      condition     = var.tiering_obs_name != ""
      error_message = "Tiering OBS is not provided"
    }
    precondition {
      condition     = var.tiering_obs_container_name != ""
      error_message = "Tiering OBS container name is not provided"
    }
  }
}

data "azurerm_storage_account_blob_container_sas" "function_app_code_sas" {
  count             = local.sa_public_access_enabled || local.sa_public_access_for_vnet && local.sa_allowed_ips_provided ? 0 : 1
  connection_string = local.deployment_sa_connection_string
  container_name    = local.deployment_container_name
  start             = timestamp()
  expiry            = formatdate("YYYY-MM-DD'T'hh:mm:ssZ", timeadd(timestamp(), "1h"))
  permissions {
    read   = true
    add    = false
    create = false
    write  = false
    delete = false
    list   = false
  }
}
