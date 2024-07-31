locals {
  deployment_storage_account_id   = var.deployment_storage_account_name == "" ? azurerm_storage_account.deployment_sa[0].id : data.azurerm_storage_account.deployment_blob[0].id
  deployment_storage_account_name = var.deployment_storage_account_name == "" ? azurerm_storage_account.deployment_sa[0].name : var.deployment_storage_account_name
  deployment_sa_connection_string = var.deployment_storage_account_name == "" ? azurerm_storage_account.deployment_sa[0].primary_connection_string : data.azurerm_storage_account.deployment_blob[0].primary_connection_string
  deployment_container_name       = var.deployment_container_name == "" ? "${local.alphanumeric_prefix_name}${local.alphanumeric_cluster_name}-deployment" : var.deployment_container_name
  deployment_sa_access_key        = var.deployment_storage_account_name == "" ? azurerm_storage_account.deployment_sa[0].primary_access_key : data.azurerm_storage_account.deployment_blob[0].primary_access_key
}

resource "azurerm_storage_account" "deployment_sa" {
  count                    = var.deployment_storage_account_name == "" && var.allow_sa_public_network_access ? 1 : 0
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
}

resource "azurerm_storage_container" "deployment" {
  count                 = var.deployment_container_name == "" && var.allow_sa_public_network_access ? 1 : 0
  name                  = "${local.alphanumeric_prefix_name}${local.alphanumeric_cluster_name}-deployment"
  storage_account_name  = local.deployment_storage_account_name
  container_access_type = "private"
  depends_on            = [azurerm_storage_account.deployment_sa]
}

resource "azurerm_storage_blob" "state" {
  count                  = var.allow_sa_public_network_access ? 1 : 0
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

# state for protocols
resource "azurerm_storage_container" "nfs_deployment" {
  count                 = var.nfs_deployment_container_name == "" && var.allow_sa_public_network_access ? 1 : 0
  name                  = "${local.alphanumeric_prefix_name}${local.alphanumeric_cluster_name}-protocol-deployment"
  storage_account_name  = local.deployment_storage_account_name
  container_access_type = "private"
  depends_on            = [azurerm_storage_account.deployment_sa]
}

resource "azurerm_storage_blob" "nfs_state" {
  count                  = var.nfs_protocol_gateways_number > 0 && var.allow_sa_public_network_access ? 1 : 0
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
  count                    = var.allow_sa_public_network_access ? 1 : 0
  name                     = substr("${local.alphanumeric_prefix_name}${local.alphanumeric_cluster_name}logicappsa", 0, 24)
  resource_group_name      = var.rg_name
  location                 = local.location
  account_tier             = "Standard"
  account_replication_type = "LRS"
}

data "azurerm_storage_account" "deployment_blob" {
  count               = var.deployment_storage_account_name != "" ? 1 : 0
  name                = var.deployment_storage_account_name
  resource_group_name = local.resource_group_name
}

resource "azurerm_private_dns_zone" "blob" {
  count               = !var.allow_sa_public_network_access && var.create_storage_account_private_links ? 1 : 0
  name                = "privatelink.blob.core.windows.net"
  resource_group_name = local.resource_group_name

  lifecycle {
    precondition {
      condition     = var.deployment_function_app_code_blob != ""
      error_message = "Function app code blob is not provided"
    }
    precondition {
      condition     = var.deployment_container_name != ""
      error_message = "Function app storage account container name is not provided"
    }
    precondition {
      condition     = var.deployment_storage_account_name != ""
      error_message = "Function app storage account name is not provided"
    }
  }
}

resource "azurerm_private_dns_zone" "file" {
  count               = !var.allow_sa_public_network_access && var.create_storage_account_private_links ? 1 : 0
  name                = "privatelink.file.core.windows.net"
  resource_group_name = local.resource_group_name

  depends_on = [azurerm_private_dns_zone.blob]
}

resource "azurerm_private_dns_zone_virtual_network_link" "blob_privatelink" {
  count                 = !var.allow_sa_public_network_access && var.create_storage_account_private_links ? 1 : 0
  name                  = "${var.prefix}-${var.cluster_name}-blob-privatelink"
  resource_group_name   = local.resource_group_name
  private_dns_zone_name = azurerm_private_dns_zone.blob[0].name
  virtual_network_id    = data.azurerm_virtual_network.vnet.id
}

resource "azurerm_private_dns_zone_virtual_network_link" "file_privatelink" {
  count                 = !var.allow_sa_public_network_access && var.create_storage_account_private_links ? 1 : 0
  name                  = "${var.prefix}-${var.cluster_name}-file-privatelink"
  resource_group_name   = local.resource_group_name
  private_dns_zone_name = azurerm_private_dns_zone.file[0].name
  virtual_network_id    = data.azurerm_virtual_network.vnet.id
}

resource "azurerm_private_endpoint" "file_endpoint" {
  count               = !var.allow_sa_public_network_access && var.create_storage_account_private_links ? 1 : 0
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
  count               = !var.allow_sa_public_network_access && var.create_storage_account_private_links ? 1 : 0
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

data "azurerm_storage_account_blob_container_sas" "function_app_code_sas" {
  count             = var.allow_sa_public_network_access ? 0 : 1
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
