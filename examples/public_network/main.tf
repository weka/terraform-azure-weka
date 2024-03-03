provider "azurerm" {
  subscription_id = var.subscription_id
  partner_id      = "f13589d1-f10d-4c3b-ae42-3b1a8337eaf1"
  features {
  }
}

module "weka_deployment" {
  source                         = "../.."
  prefix                         = "test"
  rg_name                        = "baruch-rg"
  get_weka_io_token              = var.get_weka_io_token
  subscription_id                = var.subscription_id
  cluster_name                   = "l8as"
  tiering_enable_obs_integration = true
  cluster_size                   = 6
  allow_ssh_cidrs                = ["0.0.0.0/0"]
  allow_weka_api_cidrs           = ["0.0.0.0/0"]
  instance_type = "Standard_L8as_v3"
  weka_version = "4.3.0.325-nightly"
  ssh_public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC83xywjfh32vOUZGc2cWMBI7s0krK1au2EkWSTLkkOnsW7QVulrwqT5yi+02lVsJ7TPIV0DYTyg2GjkcUoBOyTu0/Msly9cTPv033SD+17CY3WAG29kY0OGkxSugpEWp4Z+vaQqGWP0G3D7yxBXQ0m0W3yDzNV+Jk3PERh4t7VU4T+zRmGy1cBttW1nQH9BewqgNfynQvUr3/YBkQXP0g2yTWtFM+0BUv4imcNpgm4/MQyQX41PJt0ey8v/pEuz9Hl75aZINwkdbQvSVWO2pcwwtkMtSK/89kYKCI3bF0gBSUPlnoPZorYyk+Y99nrOLUhSdrC8IjZ2DLQfzuwLNtl weka_id_rsa_2019.05.27"
}
