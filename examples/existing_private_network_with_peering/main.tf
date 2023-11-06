provider "azurerm" {
  subscription_id = var.subscription_id
  partner_id      = "f13589d1-f10d-4c3b-ae42-3b1a8337eaf1"
  features {
  }
}

module "weka_deployment" {
  source                                 = "../.."
  prefix                                 = "weka"
  rg_name                                = "weka-rg"
  assign_public_ip                       = false
  apt_repo_server                        = "http://11.0.0.4/ubuntu/mirror/archive.ubuntu.com/ubuntu/"
  install_weka_url                       = "..."
  vnet_rg_name                           = "weka-rg"
  vnet_name                              = "weka-vnet"
  subnet_name                            = "weka-subnet"
  sg_id                                  = "/subscriptions/../resourceGroups/../providers/Microsoft.Network/networkSecurityGroups/.."
  cluster_name                           = "poc"
  cluster_size                           = 6
  set_obs_integration                    = true
  subscription_id                        = var.subscription_id
  weka_tar_storage_account_id            = "/subscriptions/../resourceGroups/../providers/Microsoft.Storage/storageAccounts/.."
  function_public_network_access_enabled = false
  vnet_to_peering = [{
    vnet = "ubuntu-apt-repo-vnet"
    rg   = "ubuntu-apt-repo-rg"
  }]
}
