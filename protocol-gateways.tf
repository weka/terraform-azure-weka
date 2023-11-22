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
  backend_lb_ip                = azurerm_lb.backend_lb.private_ip_address
  install_weka_url             = local.install_weka_url
  instance_type                = var.nfs_protocol_gateway_instance_type
  apt_repo_server              = var.apt_repo_server
  vm_username                  = var.vm_username
  ssh_public_key               = var.ssh_public_key == null ? tls_private_key.ssh_key[0].public_key_openssh : var.ssh_public_key
  ppg_id                       = local.placement_group_id
  sg_id                        = local.sg_id
  key_vault_url                = azurerm_key_vault.key_vault.vault_uri
  key_vault_id                 = azurerm_key_vault.key_vault.id
  assign_public_ip             = var.assign_public_ip
  disk_size                    = var.nfs_protocol_gateway_disk_size
  frontend_container_cores_num = var.nfs_protocol_gateway_fe_cores_num
  depends_on                   = [module.network, azurerm_linux_virtual_machine_scale_set.vmss, azurerm_key_vault_secret.get_weka_io_token, azurerm_proximity_placement_group.ppg]
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
  backend_lb_ip                = azurerm_lb.backend_lb.private_ip_address
  install_weka_url             = local.install_weka_url
  instance_type                = var.smb_protocol_gateway_instance_type
  apt_repo_server              = var.apt_repo_server
  vm_username                  = var.vm_username
  ssh_public_key               = var.ssh_public_key == null ? tls_private_key.ssh_key[0].public_key_openssh : var.ssh_public_key
  ppg_id                       = local.placement_group_id
  sg_id                        = local.sg_id
  key_vault_url                = azurerm_key_vault.key_vault.vault_uri
  key_vault_id                 = azurerm_key_vault.key_vault.id
  assign_public_ip             = var.assign_public_ip
  disk_size                    = var.smb_protocol_gateway_disk_size
  frontend_container_cores_num = var.smb_protocol_gateway_fe_cores_num
  smb_cluster_name             = var.smb_cluster_name
  smb_domain_name              = var.smb_domain_name
  smb_domain_netbios_name      = var.smb_domain_netbios_name
  smb_dns_ip_address           = var.smb_dns_ip_address
  smb_share_name               = var.smb_share_name
  smbw_enabled                 = var.smbw_enabled
  depends_on                   = [module.network, azurerm_linux_virtual_machine_scale_set.vmss, azurerm_key_vault_secret.get_weka_io_token, azurerm_proximity_placement_group.ppg]
}
