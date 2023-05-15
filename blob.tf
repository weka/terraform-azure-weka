data "azurerm_private_dns_zone" "blob_dns_zone" {
  name                = var.blob_dns_zone_name
  resource_group_name = var.rg_name
}

data azurerm_subnet "subnets_delegation" {
  count                = length(var.subnets_delegation_names)
  name                 = var.subnets_delegation_names[count.index]
  resource_group_name  = var.rg_name
  virtual_network_name = var.vnet_name
}

resource "azurerm_storage_account" "deployment_sa" {
  name                      = "${local.alphanumeric_prefix_name}${local.alphanumeric_cluster_name}deployment"
  location                  = local.location
  resource_group_name       = var.rg_name
  account_kind              = "StorageV2"
  account_tier              = "Standard"
  account_replication_type  = "ZRS"
  enable_https_traffic_only = true
  tags                      = merge(var.tags_map, { "weka_cluster" : var.cluster_name })
  network_rules {
    default_action             = "Deny"
    ip_rules                   = [local.my_ip]
    bypass                     = ["AzureServices", "Logging","Metrics"]
    virtual_network_subnet_ids = concat(data.azurerm_subnet.subnets.*.id, [data.azurerm_subnet.subnets_delegation[0].id])
  }
  identity {
    type = "SystemAssigned"
  }
  routing {
    choice = "MicrosoftRouting"
  }
  lifecycle {
    ignore_changes = [tags]
  }
}

resource "azurerm_storage_container" "deployment" {
  name                  = "${local.alphanumeric_prefix_name}${local.alphanumeric_cluster_name}-deployment"
  storage_account_name  = azurerm_storage_account.deployment_sa.name
  container_access_type = "private"
  depends_on            = [azurerm_storage_account.deployment_sa]
}

resource "azurerm_storage_blob" "state" {
  name                   = "state"
  storage_account_name   = azurerm_storage_account.deployment_sa.name
  storage_container_name = azurerm_storage_container.deployment.name
  type                   = "Block"
  source_content         = "{\"initial_size\":${var.cluster_size}, \"desired_size\":${var.cluster_size}, \"instances\":[], \"clusterized\":false}"
  depends_on             = [azurerm_storage_container.deployment]

  lifecycle {
    ignore_changes = all
  }
}

resource "azurerm_private_endpoint" "storage_account_endpoint" {
  name                          = "${var.prefix}-${var.cluster_name}-sa-endpoint"
  resource_group_name           = var.rg_name
  location                      = local.location
  subnet_id                     = data.azurerm_subnet.subnets[0].id
  custom_network_interface_name = "${var.prefix}-${var.cluster_name}-sa-endpoint"
  private_service_connection {
    name                           = "${var.prefix}-${var.cluster_name}-sa-endpoint"
    private_connection_resource_id = azurerm_storage_account.deployment_sa.id
    is_manual_connection           = false
    subresource_names              = ["blob"]
  }
  private_dns_zone_group {
    name                 = "${var.prefix}-${var.cluster_name}-sa-endpoint"
    private_dns_zone_ids = [data.azurerm_private_dns_zone.blob_dns_zone.id]
  }
  lifecycle {
    ignore_changes = [tags]
  }
  depends_on = [azurerm_storage_account.deployment_sa]
}