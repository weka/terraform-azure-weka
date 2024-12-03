module "aks_clients" {
  count                        = var.aks_clients ? 1 : 0
  source                       = "./modules/aks"
  rg_name                      = var.rg_name
  subnet_name                  = local.subnet_name
  vnet_name                    = local.vnet_name
  frontend_container_cores_num = var.aks_client_frontend_cores
  instance_type                = var.aks_client_instance_type
  ssh_public_key               = var.ssh_public_key == null ? tls_private_key.ssh_key[0].public_key_openssh : var.ssh_public_key
  key_vault_name               = azurerm_key_vault.key_vault.name
  prefix                       = var.prefix
  backend_vmss_name            = local.vmss_name # <prefix>-<cluster-name>-vmss
  subscription_id              = var.subscription_id
  node_count                   = var.aks_instances_number
  create_ml                    = var.create_ml
  cluster_name                 = var.cluster_name
  depends_on                   = [module.network, azurerm_linux_function_app.function_app]
}
