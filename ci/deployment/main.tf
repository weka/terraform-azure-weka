provider "azurerm" {
  subscription_id = var.subscription_id
  partner_id      = "f13589d1-f10d-4c3b-ae42-3b1a8337eaf1"
  client_id       = var.client_id
  tenant_id       = var.tenant_id
  client_secret   = var.client_secret

  features {
  }
}

module "weka_deployment" {
  source              = "../.."
  prefix              = var.prefix
  rg_name             = var.rg_name
  get_weka_io_token   = var.get_weka_io_token
  subscription_id     = var.subscription_id
  cluster_name        = var.cluster_name
  set_obs_integration = true
  cluster_size        = var.cluster_size
  tiering_ssd_percent = 20
  allow_ssh_ranges    = ["0.0.0.0/0"]
}
