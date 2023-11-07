provider "azurerm" {
  subscription_id = var.subscription_id
  partner_id      = "f13589d1-f10d-4c3b-ae42-3b1a8337eaf1"
  features {
  }
}

module "weka_deployment" {
  source                         = "../.."
  prefix                         = "weka"
  rg_name                        = "weka-rg"
  cluster_name                   = "poc"
  cluster_size                   = 6
  allow_ssh_cidrs                = ["0.0.0.0/0"]
  allow_weka_api_cidrs           = ["0.0.0.0/0"]
  subscription_id                = var.subscription_id
  get_weka_io_token              = var.get_weka_io_token
  tiering_enable_obs_integration = true
  tiering_obs_name               = "obs"
  tiering_obs_container_name     = "obs-container"
  tiering_blob_obs_access_key    = "..."
}
