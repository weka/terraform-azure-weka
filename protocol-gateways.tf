module "nfs_protocol_gateways" {
  count                        = var.nfs_protocol_gateways_number > 0 ? 1 : 0
  source                       = "./modules/protocol_gateways"
  rg_name                      = var.rg_name
  location                     = data.azurerm_resource_group.rg.location
  subnet_name                  = data.azurerm_subnet.subnet.name
  source_image_id              = var.source_image_id
  vnet_name                    = local.vnet_name
  vnet_rg_name                 = local.vnet_rg_name
  tags_map                     = var.tags_map
  setup_protocol               = var.nfs_setup_protocol
  gateways_number              = var.nfs_protocol_gateways_number
  gateways_name                = "${var.prefix}-${var.cluster_name}-nfs-protocol-gateway"
  protocol                     = "NFS"
  secondary_ips_per_nic        = var.nfs_protocol_gateway_secondary_ips_per_nic
  instance_type                = var.nfs_protocol_gateway_instance_type
  apt_repo_server              = var.apt_repo_server
  vm_username                  = var.vm_username
  ssh_public_key               = var.ssh_public_key == null ? tls_private_key.ssh_key[0].public_key_openssh : var.ssh_public_key
  ppg_id                       = local.placement_group_id
  sg_id                        = local.sg_id
  key_vault_url                = azurerm_key_vault.key_vault.vault_uri
  key_vault_id                 = azurerm_key_vault.key_vault.id
  assign_public_ip             = local.assign_public_ip
  traces_per_frontend          = var.traces_per_ionode
  disk_size                    = var.nfs_protocol_gateway_disk_size
  frontend_container_cores_num = var.nfs_protocol_gateway_fe_cores_num
  vm_identity_name             = var.protocol_gateways_identity_name
  deploy_function_url          = "https://${azurerm_linux_function_app.function_app.name}.azurewebsites.net/api/deploy"
  report_function_url          = "https://${azurerm_linux_function_app.function_app.name}.azurewebsites.net/api/report"
  function_app_default_key     = data.azurerm_function_app_host_keys.function_keys.default_function_key
  depends_on                   = [module.network, azurerm_key_vault_secret.get_weka_io_token, azurerm_proximity_placement_group.ppg]
}

resource "azurerm_subnet" "dns_resolver_subnet" {
  count                = var.smb_create_private_dns_resolver && var.smb_dns_resolver_subnet_delegation_id == "" ? 1 : 0
  name                 = "${var.prefix}-${var.cluster_name}-subnet-dns-resolver"
  resource_group_name  = local.resource_group_name
  virtual_network_name = local.vnet_name
  address_prefixes     = [var.smb_dns_resolver_subnet_delegation_cidr]

  delegation {
    name = "Microsoft.Network.dnsResolvers"
    service_delegation {
      actions = ["Microsoft.Network/virtualNetworks/subnets/join/action"]
      name    = "Microsoft.Network/dnsResolvers"
    }
  }
  depends_on = [module.network]
}

resource "azurerm_private_dns_resolver" "dns_resolver" {
  count               = var.smb_create_private_dns_resolver ? 1 : 0
  name                = "${var.prefix}-${var.cluster_name}-resolver"
  resource_group_name = local.resource_group_name
  location            = local.location
  virtual_network_id  = data.azurerm_virtual_network.vnet.id
  depends_on          = [module.network]
}

resource "azurerm_private_dns_resolver_outbound_endpoint" "outbound_endpoint" {
  count                   = var.smb_create_private_dns_resolver ? 1 : 0
  name                    = "${var.prefix}-${var.cluster_name}-endpoint"
  private_dns_resolver_id = azurerm_private_dns_resolver.dns_resolver[0].id
  location                = azurerm_private_dns_resolver.dns_resolver[0].location
  subnet_id               = var.smb_dns_resolver_subnet_delegation_id == "" ? azurerm_subnet.dns_resolver_subnet[0].id : var.smb_dns_resolver_subnet_delegation_id
  tags                    = merge(var.tags_map, { "weka_cluster" : var.cluster_name })
  depends_on              = [module.network, azurerm_private_dns_resolver.dns_resolver, azurerm_subnet.dns_resolver_subnet]
}

resource "azurerm_private_dns_resolver_dns_forwarding_ruleset" "dns_forwarding_ruleset" {
  count                                      = var.smb_create_private_dns_resolver ? 1 : 0
  name                                       = "${var.prefix}-${var.cluster_name}-ruleset"
  resource_group_name                        = local.resource_group_name
  location                                   = local.location
  private_dns_resolver_outbound_endpoint_ids = [azurerm_private_dns_resolver_outbound_endpoint.outbound_endpoint[0].id]
  tags                                       = merge(var.tags_map, { "weka_cluster" : var.cluster_name })
  depends_on                                 = [module.network, azurerm_private_dns_resolver_outbound_endpoint.outbound_endpoint]
}

resource "azurerm_private_dns_resolver_forwarding_rule" "resolver_forwarding_rule" {
  count                     = var.smb_create_private_dns_resolver ? 1 : 0
  name                      = "${var.prefix}-${var.cluster_name}-rule"
  dns_forwarding_ruleset_id = azurerm_private_dns_resolver_dns_forwarding_ruleset.dns_forwarding_ruleset[0].id
  domain_name               = "${var.smb_domain_name}."
  enabled                   = true
  target_dns_servers {
    ip_address = var.smb_dns_ip_address
    port       = 53
  }
  depends_on = [azurerm_private_dns_resolver_dns_forwarding_ruleset.dns_forwarding_ruleset]
}

resource "azurerm_private_dns_resolver_virtual_network_link" "dns_forwarding_virtual_network_link" {
  count                     = var.smb_create_private_dns_resolver ? 1 : 0
  name                      = "${var.prefix}-${var.cluster_name}-dns-forward-vnet-link"
  virtual_network_id        = data.azurerm_virtual_network.vnet.id
  dns_forwarding_ruleset_id = azurerm_private_dns_resolver_dns_forwarding_ruleset.dns_forwarding_ruleset[0].id
  depends_on                = [azurerm_private_dns_resolver_dns_forwarding_ruleset.dns_forwarding_ruleset]
}

module "smb_protocol_gateways" {
  count                        = var.smb_protocol_gateways_number > 0 ? 1 : 0
  source                       = "./modules/protocol_gateways"
  rg_name                      = var.rg_name
  location                     = data.azurerm_resource_group.rg.location
  subnet_name                  = data.azurerm_subnet.subnet.name
  source_image_id              = var.source_image_id
  vnet_name                    = local.vnet_name
  vnet_rg_name                 = local.vnet_rg_name
  setup_protocol               = var.smb_setup_protocol
  tags_map                     = var.tags_map
  gateways_number              = var.smb_protocol_gateways_number
  gateways_name                = "${var.prefix}-${var.cluster_name}-smb-protocol-gateway"
  protocol                     = "SMB"
  secondary_ips_per_nic        = var.smb_protocol_gateway_secondary_ips_per_nic
  instance_type                = var.smb_protocol_gateway_instance_type
  apt_repo_server              = var.apt_repo_server
  vm_username                  = var.vm_username
  ssh_public_key               = var.ssh_public_key == null ? tls_private_key.ssh_key[0].public_key_openssh : var.ssh_public_key
  ppg_id                       = local.placement_group_id
  sg_id                        = local.sg_id
  key_vault_url                = azurerm_key_vault.key_vault.vault_uri
  key_vault_id                 = azurerm_key_vault.key_vault.id
  assign_public_ip             = local.assign_public_ip
  traces_per_frontend          = var.traces_per_ionode
  disk_size                    = var.smb_protocol_gateway_disk_size
  frontend_container_cores_num = var.smb_protocol_gateway_fe_cores_num
  smb_cluster_name             = var.smb_cluster_name
  smb_domain_name              = var.smb_domain_name
  smbw_enabled                 = var.smbw_enabled
  deploy_function_url          = "https://${azurerm_linux_function_app.function_app.name}.azurewebsites.net/api/deploy"
  report_function_url          = "https://${azurerm_linux_function_app.function_app.name}.azurewebsites.net/api/report"
  function_app_default_key     = data.azurerm_function_app_host_keys.function_keys.default_function_key
  depends_on                   = [module.network, azurerm_key_vault_secret.get_weka_io_token, azurerm_proximity_placement_group.ppg, azurerm_private_dns_resolver_dns_forwarding_ruleset.dns_forwarding_ruleset]
}

module "s3_protocol_gateways" {
  count                        = var.s3_protocol_gateways_number > 0 ? 1 : 0
  source                       = "./modules/protocol_gateways"
  rg_name                      = var.rg_name
  location                     = data.azurerm_resource_group.rg.location
  subnet_name                  = data.azurerm_subnet.subnet.name
  source_image_id              = var.source_image_id
  vnet_name                    = local.vnet_name
  vnet_rg_name                 = local.vnet_rg_name
  setup_protocol               = var.s3_setup_protocol
  tags_map                     = var.tags_map
  gateways_number              = var.s3_protocol_gateways_number
  gateways_name                = "${var.prefix}-${var.cluster_name}-s3-protocol-gateway"
  protocol                     = "S3"
  secondary_ips_per_nic        = 0
  instance_type                = var.s3_protocol_gateway_instance_type
  apt_repo_server              = var.apt_repo_server
  vm_username                  = var.vm_username
  ssh_public_key               = var.ssh_public_key == null ? tls_private_key.ssh_key[0].public_key_openssh : var.ssh_public_key
  ppg_id                       = local.placement_group_id
  sg_id                        = local.sg_id
  key_vault_url                = azurerm_key_vault.key_vault.vault_uri
  key_vault_id                 = azurerm_key_vault.key_vault.id
  assign_public_ip             = local.assign_public_ip
  traces_per_frontend          = var.traces_per_ionode
  disk_size                    = var.s3_protocol_gateway_disk_size
  frontend_container_cores_num = var.s3_protocol_gateway_fe_cores_num
  deploy_function_url          = "https://${azurerm_linux_function_app.function_app.name}.azurewebsites.net/api/deploy"
  report_function_url          = "https://${azurerm_linux_function_app.function_app.name}.azurewebsites.net/api/report"
  function_app_default_key     = data.azurerm_function_app_host_keys.function_keys.default_function_key
  depends_on                   = [module.network, azurerm_key_vault_secret.get_weka_io_token, azurerm_proximity_placement_group.ppg, azurerm_private_dns_resolver_dns_forwarding_ruleset.dns_forwarding_ruleset]
}
