module "data_services" {
  count                       = var.data_services_number > 0 ? 1 : 0
  source                      = "./modules/data_services"
  rg_name                     = var.rg_name
  location                    = data.azurerm_resource_group.rg.location
  subnet_name                 = data.azurerm_subnet.subnet.name
  source_image_id             = local.source_image_id
  vnet_name                   = local.vnet_name
  vnet_rg_name                = local.vnet_rg_name
  tags_map                    = var.tags_map
  cluster_name                = var.cluster_name
  data_services_number        = var.data_services_number
  data_services_name          = "${var.prefix}-${var.cluster_name}-data-services"
  instance_type               = var.data_services_instance_type
  disk_size                   = var.data_services_disk_size
  root_volume_size            = var.data_services_root_volume_size
  apt_repo_server             = var.apt_repo_server
  vm_username                 = var.vm_username
  ssh_public_key              = var.ssh_public_key == null ? tls_private_key.ssh_key[0].public_key_openssh : var.ssh_public_key
  sg_id                       = local.sg_id
  key_vault_id                = azurerm_key_vault.key_vault.id
  assign_public_ip            = local.assign_public_ip
  vm_identity_name            = var.data_services_identity_name
  weka_tar_storage_account_id = var.weka_tar_storage_account_id
  deploy_function_url         = "https://${azurerm_linux_function_app.function_app.name}.azurewebsites.net/api/deploy"
  report_function_url         = "https://${azurerm_linux_function_app.function_app.name}.azurewebsites.net/api/report"
  function_app_default_key    = data.azurerm_function_app_host_keys.function_keys.default_function_key
  image_sku                   = var.image_sku
  image_offer                 = var.image_offer
  image_version               = var.image_version
  depends_on                  = [module.network, azurerm_key_vault_secret.get_weka_io_token]
}
