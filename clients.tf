resource "null_resource" "get-backend-ip" {
  count = var.clients_number > 0 ? 1 : 0
  triggers = {
    always_run = timestamp()
  }
  provisioner "local-exec" {
    command = "az vmss nic list -g ${var.rg_name} --vmss-name ${azurerm_linux_virtual_machine_scale_set.vmss.name} --subscription ${var.subscription_id} --query \"[].ipConfigurations[]\" | jq -r '.[] | select(.name==\"ipconfig0\")'.privateIPAddress > ${path.root}/backend_ips"
  }
  depends_on = [azurerm_linux_virtual_machine_scale_set.vmss]
}

data "local_file" "backend_ips" {
  count      = var.clients_number > 0 ? 1 : 0
  filename   = "${path.root}/backend_ips"
  depends_on = [null_resource.get-backend-ip]
}

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
  backend_ips        = [replace(join(" ",[data.local_file.backend_ips[0].content]), "\n", " ")]
  ssh_public_key     = var.ssh_public_key == null ? tls_private_key.ssh_key[0].public_key_openssh : var.ssh_public_key
  ppg_id             = local.placement_group_id
  assign_public_ip   = var.private_network ? false : true
  vnet_rg_name       = var.vnet_rg_name
  source_image_id    = var.source_image_id
  sg_id              = var.sg_id
}

resource "null_resource" "clean" {
  count = var.clients_number > 0 ? 1 : 0
  triggers = {
    always_run = timestamp()
  }
  provisioner "local-exec" {
    command = "rm -f ${path.root}/backend_ips"
  }
  depends_on = [module.clients]
}