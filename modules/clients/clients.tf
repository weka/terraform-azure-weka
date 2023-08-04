data "azurerm_resource_group" "rg" {
  name = var.rg_name
}

data "azurerm_subnet" "subnet" {
  resource_group_name  = var.vnet_rg_name
  virtual_network_name = var.vnet_name
  name                 = var.subnet_name
}

locals {
  private_nic_first_index = var.assign_public_ip ? 1 : 0
  preparation_script      = templatefile("${path.module}/init.sh", {
    apt_repo_server = var.apt_repo_server
    nics_num        = var.nics
    subnet_range    = data.azurerm_subnet.subnet.address_prefix
  })

  mount_wekafs_script = templatefile("${path.module}/mount_wekafs.sh", {
    all_subnets = split("/", data.azurerm_subnet.subnet.address_prefix)[0]
    all_gateways = cidrhost(data.azurerm_subnet.subnet.address_prefix, 1)
    nics_num           = var.nics
    backend_lb_ip      = var.backend_lb_ip
    mount_clients_dpdk = var.mount_clients_dpdk
  })

  custom_data_parts = [local.preparation_script, local.mount_wekafs_script]
  vms_custom_data   = base64encode(join("\n", local.custom_data_parts))
}

resource "azurerm_linux_virtual_machine_scale_set" "vmss" {
  name                            = "${var.clients_name}-vmss"
  location                        = data.azurerm_resource_group.rg.location
  resource_group_name             = var.rg_name
  sku                             = var.instance_type
  upgrade_mode                    = "Manual"
  admin_username                  = var.vm_username
  instances                       = var.clients_number
  computer_name_prefix            = "${var.clients_name}-vmss"
  custom_data                     = local.vms_custom_data
  disable_password_authentication = true
  proximity_placement_group_id    = var.ppg_id
  tags                            = merge( {"weka_cluster": var.clients_name})
  source_image_id                 = var.source_image_id

  os_disk {
    caching              = "ReadWrite"
    storage_account_type = "StandardSSD_LRS"
  }

  admin_ssh_key {
    username   = var.vm_username
    public_key = var.ssh_public_key
  }

  identity {
    type = "SystemAssigned"
  }

  dynamic "network_interface" {
    for_each = range(local.private_nic_first_index)
    content {
      name                          = "${var.clients_name}-nic-${network_interface.value}"
      network_security_group_id     = var.sg_id
      primary                       = true
      enable_accelerated_networking = true
      ip_configuration {
        primary                                = true
        name                                   = "ipconfig0"
        subnet_id                              = data.azurerm_subnet.subnet.id
        public_ip_address {
          name = "${var.clients_name}-public-ip"
        }
      }
    }
  }
  dynamic "network_interface" {
    for_each = range(local.private_nic_first_index, 1)
    content {
      name                          = "${var.clients_name}-nic-${network_interface.value}"
      network_security_group_id     = var.sg_id
      primary                       = true
      enable_accelerated_networking = true
      ip_configuration {
        primary    = true
        name       = "ipconfig0"
        subnet_id  = data.azurerm_subnet.subnet.id
      }
    }
  }
  dynamic "network_interface" {
    for_each = range(1, var.nics)
    content {
      name                          = "${var.clients_name}-nic-${network_interface.value}"
      network_security_group_id     = var.sg_id
      primary                       = false
      enable_accelerated_networking = true
      ip_configuration {
        primary       = true
        name          = "ipconfig-${network_interface.value}"
        subnet_id     = data.azurerm_subnet.subnet.id
      }
    }
  }
  lifecycle {
    ignore_changes = [ instances, custom_data ]
  }
}
