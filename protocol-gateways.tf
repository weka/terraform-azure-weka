resource "null_resource" "backend_ips_for_protocol_gateways" {
  count = var.protocol_gateways_number > 0 ? 1 : 0
  triggers = {
    always_run = timestamp()
  }
  provisioner "local-exec" {
    command = "az vmss nic list -g ${var.rg_name} --vmss-name ${azurerm_linux_virtual_machine_scale_set.vmss.name} --subscription ${var.subscription_id} --query \"[].ipConfigurations[]\" | jq -r '.[] | select(.name==\"ipconfig0\")'.privateIPAddress > ${path.root}/backend_ips_for_protocol_gateways"
  }
  depends_on = [azurerm_linux_virtual_machine_scale_set.vmss]
}

data "local_file" "backend_ips_for_protocol_gateways" {
  count      = var.protocol_gateways_number > 0 ? 1 : 0
  filename   = "${path.root}/backend_ips_for_protocol_gateways"
  depends_on = [null_resource.backend_ips_for_protocol_gateways]
}

module "protocol_gateways" {
  count                      = var.protocol_gateways_number > 0 ? 1 : 0
  
  source                     = "./modules/protocol_gateways"
  rg_name                    = var.rg_name
  subnet_name                = data.azurerm_subnet.subnet.name
  source_image_id            = var.source_image_id
  vnet_name                  = var.vnet_name
  vnet_rg_name               = var.vnet_rg_name
  tags_map                   = var.tags_map
  gateways_number            = var.protocol_gateways_number
  gateways_name              = "${var.prefix}-${var.cluster_name}-protocol-gateway"
  protocol                   = var.protocol
  nics                       = var.protocol_gateway_nics_num
  secondary_ips_per_nic      = var.protocol_gateway_secondary_ips_per_nic
  backend_ips                = compact(split("\n", data.local_file.backend_ips_for_protocol_gateways[0].content))
  install_weka_url           = var.install_weka_url != "" ? var.install_weka_url : "https://$TOKEN@get.weka.io/dist/v1/install/${var.weka_version}/${var.weka_version}"
  instance_type              = var.protocol_gateway_instance_type
  apt_repo_url               = var.apt_repo_url
  vm_username                = var.vm_username
  ssh_public_key             = var.ssh_public_key == null ? tls_private_key.ssh_key[0].public_key_openssh : var.ssh_public_key
  ppg_id                     = local.placement_group_id
  sg_id                      = var.sg_id
  key_vault_url              = azurerm_key_vault.key_vault.vault_uri
  assign_public_ip           = var.private_network ? false : true
  disk_size                  = var.protocol_gateway_disk_size
  frontend_num               = var.protocol_gateway_frontend_num

  depends_on = [azurerm_linux_virtual_machine_scale_set.vmss, azurerm_key_vault_secret.get_weka_io_token]
}

resource "null_resource" "clean_backend_ips_for_protocol_gateways" {
  count = var.protocol_gateways_number > 0 ? 1 : 0
  triggers = {
    always_run = timestamp()
  }
  provisioner "local-exec" {
    command = "rm -f ${path.root}/backend_ips_for_protocol_gateways"
  }
  depends_on = [module.protocol_gateways]
}

resource "azurerm_key_vault_access_policy" "gateways_vmss_key_vault" {
  count        = var.protocol_gateways_number > 0 ? 1 : 0

  key_vault_id = azurerm_key_vault.key_vault.id
  tenant_id    = data.azurerm_client_config.current.tenant_id
  object_id    = module.protocol_gateways[0].vmss_principal_id
  secret_permissions = [
    "Get",
  ]
  depends_on = [azurerm_key_vault.key_vault, module.protocol_gateways]
}

resource "azurerm_role_assignment" "gateways_vmss_key_vault" {
  count                = var.protocol_gateways_number > 0 ? 1 : 0

  scope                = azurerm_key_vault.key_vault.id
  role_definition_name = "Key Vault Secrets User"
  principal_id         = module.protocol_gateways[0].vmss_principal_id
  depends_on           = [azurerm_key_vault.key_vault, module.protocol_gateways]
}
