resource "azurerm_storage_account" "deployment_sa" {
  name                     = "${local.alphanumeric_prefix_name}${local.alphanumeric_cluster_name}deployment"
  location                 = data.azurerm_resource_group.rg.location
  resource_group_name      = var.rg_name
  account_kind             = "StorageV2"
  account_tier             = "Standard"
  account_replication_type = "ZRS"
  tags                     = merge(var.tags_map, { "weka_cluster" : var.cluster_name })
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
