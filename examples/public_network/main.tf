provider "azurerm" {
  subscription_id = var.subscription_id
  partner_id      = "f13589d1-f10d-4c3b-ae42-3b1a8337eaf1"
  features {
  }
}


module "weka_deployment" {
  source                              = "../.."
  prefix                              = "aks"
  rg_name                             = "denise"
  get_weka_io_token                   = var.get_weka_io_token
  subscription_id                     = var.subscription_id
  cluster_name                        = "ml"
  tiering_enable_obs_integration      = true
  cluster_size                        = 6
  allow_ssh_cidrs                     = ["0.0.0.0/0"]
  allow_weka_api_cidrs                = ["0.0.0.0/0"]
  assign_public_ip                    = true
  address_space                       = "10.224.0.0/12"
  subnet_prefix                       = "10.224.0.0/16"
  function_app_subnet_delegation_cidr = "10.225.1.0/24"
  logic_app_subnet_delegation_cidr    = "10.225.2.0/24"
  aks_clients                         = true
  aks_instances_number                = 3
  aks_client_frontend_cores           = 1
  create_ml                           = true
}
