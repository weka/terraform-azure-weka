data "azurerm_resource_group" "rg" {
  name = var.rg_name
}

locals {
  obs_scope                        = var.tiering_obs_name != "" ? "${data.azurerm_storage_account.obs_sa[0].id}/blobServices/default/containers/${var.obs_container_name}" : ""
  deployment_storage_account_scope = "${var.deployment_storage_account_id}/blobServices/default/containers/${var.deployment_container_name}"
  nfs_deployment_sa_scope          = "${var.deployment_storage_account_id}/blobServices/default/containers/${var.nfs_deployment_container_name}"
}

data "azurerm_storage_account" "obs_sa" {
  count               = var.tiering_obs_name != "" ? 1 : 0
  name                = var.tiering_obs_name
  resource_group_name = var.rg_name
}
