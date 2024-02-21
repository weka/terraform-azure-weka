locals {
  default_client_instance_type = {
    x86_64 = "Standard_D8_v5"
    arm64  = "Standard_E2ps_v5"
  }
}


module "clients" {
  count                        = var.clients_number > 0 ? 1 : 0
  source                       = "./modules/clients"
  rg_name                      = var.rg_name
  clients_name                 = "${var.prefix}-${var.cluster_name}-client"
  clients_number               = var.clients_number
  clients_use_dpdk             = var.clients_use_dpdk
  subnet_name                  = local.subnet_name
  apt_repo_server              = var.apt_repo_server
  vnet_name                    = local.vnet_name
  frontend_container_cores_num = var.clients_use_dpdk ? var.client_frontend_cores : 1
  instance_type                = var.client_instance_type != "" ? var.client_instance_type : local.default_client_instance_type[var.client_arch]
  backend_lb_ip                = azurerm_lb.backend_lb.private_ip_address
  ssh_public_key               = var.ssh_public_key == null ? tls_private_key.ssh_key[0].public_key_openssh : var.ssh_public_key
  ppg_id                       = var.client_placement_group_id == "" ? local.placement_group_id : var.client_placement_group_id
  assign_public_ip             = local.assign_public_ip
  vnet_rg_name                 = local.vnet_rg_name
  source_image_id              = var.client_source_image_id[var.client_arch]
  sg_id                        = local.sg_id
  tags_map                     = var.tags_map
  custom_data                  = var.clients_custom_data
  depends_on                   = [azurerm_proximity_placement_group.ppg, module.network]
}
