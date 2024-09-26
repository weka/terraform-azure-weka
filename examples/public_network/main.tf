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
  get_weka_io_token              = var.get_weka_io_token
  subscription_id                = var.subscription_id
  cluster_name                   = "poc"
  tiering_enable_obs_integration = true
  cluster_size                   = 6
  allow_ssh_cidrs                = ["0.0.0.0/0"]
  allow_weka_api_cidrs           = ["0.0.0.0/0"]
  assign_public_ip               = true
  weka_version                   = "4.2.14"
  source_image_id                = "/communityGalleries/WekaIO-ddbef83d-dec1-42d0-998a-3c083f1450b7/images/weka_custom_image_nvme/versions/1.0.0"
  instance_type                  = "Standard_L16aos_v4"
}
