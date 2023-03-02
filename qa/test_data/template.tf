provider "azurerm" {
  subscription_id = local.subscription_id
  client_id       = local.client_id
  tenant_id       = local.tenant_id
  client_secret   = local.client_secret
  features {
    resource_group {
      prevent_deletion_if_contains_resources = false
    }
  }
}

# Create resource group

resource "azurerm_resource_group" "autotest_rg" {
  location = local.location
  name     = local.rg_name
}

# Create network

module "create-network" {
  source          = "../../../modules/create_networks"
  prefix          = local.prefix
  rg_name         = azurerm_resource_group.autotest_rg.name
  address_space   = local.address_space
  subnet_delegation = local.subnet_delegation
  subnet_prefixes = local.subnet_prefixes
  depends_on      = [azurerm_resource_group.autotest_rg]
}

# Deploy Weka

module "deploy-weka" {
  source                = "../../.."
  prefix                = local.prefix
  rg_name               = azurerm_resource_group.autotest_rg.name
  vnet_name             = module.create-network.vnet-name
  subnets               = module.create-network.subnets-name
  sg_id                 = module.create-network.sg-id
  get_weka_io_token     = local.get_weka_io_token
  cluster_name          = local.cluster_name
  subnet_delegation_id  = module.create-network.subnet-delegation-id
  set_obs_integration   = local.set_obs_integration
  instance_type         = local.instance_type
  cluster_size          = local.cluster_size
  tiering_ssd_percent   = local.tiering_ssd_percent
  subscription_id       = local.subscription_id
  private_dns_zone_name = module.create-network.private-dns-zone-name
  depends_on            = [module.create-network, azurerm_resource_group.autotest_rg]
}

output "get-cluster-helpers-commands" {
  value = module.deploy-weka.cluster_helpers_commands
}


