locals {
  clusterization_target = var.clusterization_target != null ? var.clusterization_target : min(var.cluster_size, max(20, ceil(var.cluster_size * 0.8)))
}


resource "azurerm_storage_account" "deployment_sa" {
  count                    = var.deployment_storage_account_name == "" ? 1 : 0
  name                     = "${local.alphanumeric_prefix_name}${local.alphanumeric_cluster_name}deployment"
  location                 = data.azurerm_resource_group.rg.location
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
  count                 = var.deployment_container_name == "" ? 1 : 0
  name                  = "${local.alphanumeric_prefix_name}${local.alphanumeric_cluster_name}-deployment"
  storage_account_name  = local.deployment_storage_account_name
  container_access_type = "private"
  depends_on            = [azurerm_storage_account.deployment_sa]
}

resource "azurerm_storage_blob" "state" {
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

data "azurerm_storage_account" "deployment_blob" {
  count               = var.deployment_storage_account_name != "" ? 1 : 0
  name                = var.deployment_storage_account_name
  resource_group_name = var.rg_name
}

data "azurerm_storage_account" "obs_sa" {
  count               = var.tiering_obs_name != "" ? 1 : 0
  name                = var.tiering_obs_name
  resource_group_name = var.rg_name
}
