module "clients" {
  count              = var.clients_number > 0 ? 1 : 0
  source             = "./modules/clients"
  rg_name            = var.rg_name
  clients_name       = "${var.prefix}-${var.cluster_name}-client"
  clients_number     = var.clients_number
  mount_clients_dpdk = var.mount_clients_dpdk
  subnet_name        = var.subnet_name
  apt_repo_url       = var.apt_repo_url
  vnet_name          = var.vnet_name
  nics               = var.mount_clients_dpdk ? var.client_nics_num : 1
  instance_type      = var.client_instance_type
  backend_lb_ip      = azurerm_lb.backend-lb.private_ip_address
  ssh_public_key     = var.ssh_public_key == null ? tls_private_key.ssh_key[0].public_key_openssh : var.ssh_public_key
  ppg_id             = local.placement_group_id
  assign_public_ip   = var.private_network ? false : true
  vnet_rg_name       = var.vnet_rg_name
  source_image_id    = var.source_image_id
  sg_id              = var.sg_id
  depends_on         = [azurerm_proximity_placement_group.ppg]
}
