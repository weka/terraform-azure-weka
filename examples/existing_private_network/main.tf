provider "azurerm" {
  subscription_id = var.subscription_id
  partner_id      = "f13589d1-f10d-4c3b-ae42-3b1a8337eaf1"
  features {
  }
}

module "weka_deployment" {
  source                                 = "../.."
  prefix                                 = "weka"
  rg_name                                = "rg-name"
  cluster_name                           = "poc"
  subnet_name                            = "subnet-name"
  vnet_name                              = "vnet-name"
  vnet_rg_name                           = "vnet-rg-name"
  sg_id                                  = "/subscriptions/../resourceGroups/../providers/Microsoft.Network/networkSecurityGroups/.."
  install_weka_url                       = "https://wekatars.blob.core.windows.net/tars/weka-4.2.5.tar"
  assign_public_ip                       = false
  cluster_size                           = 6
  set_obs_integration                    = true
  subscription_id                        = var.subscription_id
  weka_tar_storage_account_id            = "/subscriptions/../resourceGroups/../providers/Microsoft.Storage/storageAccounts/.."
  function_public_network_access_enabled = false
}
