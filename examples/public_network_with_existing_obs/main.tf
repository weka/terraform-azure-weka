provider "azurerm" {
  subscription_id = var.subscription_id
  partner_id      = "f13589d1-f10d-4c3b-ae42-3b1a8337eaf1"
  features {
  }
}

module "weka_deployment" {
  source                = "../.."
  prefix                = "weka"
  rg_name               = "weka-rg"
  cluster_name          = "poc"
  cluster_size          = 6
  allow_ssh_ranges      = ["0.0.0.0/0"]
  subscription_id       = var.subscription_id
  get_weka_io_token     = var.get_weka_io_token
  set_obs_integration   = true
  obs_name              = "obs"
  obs_container_name    = "obs-container"
  blob_obs_access_key   = "..."
  tiering_ssd_percent   = 20
}