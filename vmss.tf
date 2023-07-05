data "azurerm_resource_group" "rg" {
  name = var.rg_name
}

data "azurerm_subnet" "subnet" {
  resource_group_name  = var.vnet_rg_name
  virtual_network_name = var.vnet_name
  name                 = var.subnet_name
}

data "azurerm_virtual_network" "vnet" {
  name                = var.vnet_name
  resource_group_name = var.vnet_rg_name
}

# ===================== SSH key ++++++++++++++++++++++++= #
resource "tls_private_key" "ssh_key" {
  count     = var.ssh_public_key == null ? 1 : 0
  algorithm = "RSA"
  rsa_bits  = 4096
}

resource "local_file" "public_key" {
  count           = var.ssh_public_key == null ? 1 : 0
  content         = tls_private_key.ssh_key[count.index].public_key_openssh
  filename        = "${local.ssh_path}-public-key.pub"
  file_permission = "0600"
}

resource "local_file" "private_key" {
  count           = var.ssh_public_key == null ? 1 : 0
  content         = tls_private_key.ssh_key[count.index].private_key_pem
  filename        = "${local.ssh_path}-private-key.pem"
  file_permission = "0600"
}

locals {
  ssh_path                  = "/tmp/${var.prefix}-${var.cluster_name}"
  public_ssh_key            = var.ssh_public_key == null ? tls_private_key.ssh_key[0].public_key_openssh : file(var.ssh_public_key)
  private_ssh_key           = var.ssh_private_key == null ? tls_private_key.ssh_key[0].private_key_pem : file(var.ssh_private_key)
  disk_size                 = var.default_disk_size + var.traces_per_ionode * (var.container_number_map[var.instance_type].compute + var.container_number_map[var.instance_type].drive + var.container_number_map[var.instance_type].frontend)
  private_nic_first_index   = var.private_network ? 0 : 1
  alphanumeric_cluster_name = lower(replace(var.cluster_name, "/\\W|_|\\s/", ""))
  alphanumeric_prefix_name  = lower(replace(var.prefix, "/\\W|_|\\s/", ""))
  subnet_range              = data.azurerm_subnet.subnet.address_prefix
  nics_numbers              = var.install_cluster_dpdk ? var.container_number_map[var.instance_type].nics : 1
  custom_data_script        = templatefile("${path.module}/user-data.sh", {
    apt_repo_url             = var.apt_repo_url
    private_ssh_key          = local.private_ssh_key
    user                     = var.vm_username
    install_cluster_dpdk     = var.install_cluster_dpdk
    subnet_range             = local.subnet_range
    nics_num                 = local.nics_numbers
    deploy_url               = "https://${var.prefix}-${var.cluster_name}-function-app.azurewebsites.net/api/deploy"
    report_url               = "https://${var.prefix}-${var.cluster_name}-function-app.azurewebsites.net/api/report"
    function_app_default_key = data.azurerm_function_app_host_keys.function_keys.default_function_key
    disk_size                = local.disk_size
  })
  placement_group_id = var.placement_group_id != "" ? var.placement_group_id : azurerm_proximity_placement_group.ppg[0].id
}

resource "azurerm_proximity_placement_group" "ppg" {
  count               = var.placement_group_id == "" ? 1 : 0
  name                = "${var.prefix}-${var.cluster_name}-backend-ppg"
  location            = data.azurerm_resource_group.rg.location
  resource_group_name = var.rg_name
  tags                = merge(var.tags_map, { "weka_cluster" : var.cluster_name })
  lifecycle {
    ignore_changes = [tags]
  }
}

resource "azurerm_linux_virtual_machine_scale_set" "vmss" {
  name                            = "${var.prefix}-${var.cluster_name}-vmss"
  location                        = data.azurerm_resource_group.rg.location
  resource_group_name             = var.rg_name
  sku                             = var.instance_type
  upgrade_mode                    = "Manual"
  health_probe_id                 = azurerm_lb_probe.backend_lb_probe.id
  admin_username                  = var.vm_username
  instances                       = var.cluster_size
  computer_name_prefix            = "${var.prefix}-${var.cluster_name}-backend"
  custom_data                     = base64encode(local.custom_data_script)
  disable_password_authentication = true
  proximity_placement_group_id    = local.placement_group_id
  source_image_id                 = var.source_image_id
  tags                            = merge(var.tags_map, {
    "weka_cluster" : var.cluster_name, "user_id" : data.azurerm_client_config.current.object_id
  })

  os_disk {
    caching              = "ReadWrite"
    storage_account_type = "StandardSSD_LRS"
  }
  data_disk {
    lun                  = 0
    caching              = "ReadWrite"
    create_option        = "Empty"
    disk_size_gb         = local.disk_size
    storage_account_type = "StandardSSD_LRS"
  }

  admin_ssh_key {
    username   = var.vm_username
    public_key = local.public_ssh_key
  }

  identity {
    type = "SystemAssigned"
  }

  dynamic "network_interface" {
    for_each = range(local.private_nic_first_index)
    content {
      name                          = "${var.prefix}-${var.cluster_name}-backend-nic-0"
      network_security_group_id     = var.sg_id
      primary                       = true
      enable_accelerated_networking = var.install_cluster_dpdk
      ip_configuration {
        primary                                = true
        name                                   = "ipconfig0"
        subnet_id                              = data.azurerm_subnet.subnet.id
        load_balancer_backend_address_pool_ids = [azurerm_lb_backend_address_pool.lb_backend_pool.id]
        public_ip_address {
          name              = "${var.prefix}-${var.cluster_name}-public-ip"
          domain_name_label = "${var.prefix}-${var.cluster_name}-backend"
        }
      }
    }
  }
  dynamic "network_interface" {
    for_each = range(local.private_nic_first_index, 1)
    content {
      name                          = "${var.prefix}-${var.cluster_name}-backend-nic-0"
      network_security_group_id     = var.sg_id
      primary                       = true
      enable_accelerated_networking = var.install_cluster_dpdk
      ip_configuration {
        primary                                = true
        name                                   = "ipconfig0"
        subnet_id                              = data.azurerm_subnet.subnet.id
        load_balancer_backend_address_pool_ids = [azurerm_lb_backend_address_pool.lb_backend_pool.id]
      }
    }
  }
  dynamic "network_interface" {
    for_each = range(1, local.nics_numbers)
    content {
      name                          = "${var.prefix}-${var.cluster_name}-backend-nic-${network_interface.value}"
      network_security_group_id     = var.sg_id
      primary                       = false
      enable_accelerated_networking = var.install_cluster_dpdk
      ip_configuration {
        primary                                = false
        name                                   = "ipconfig${network_interface.value}"
        subnet_id                              = data.azurerm_subnet.subnet.id
        load_balancer_backend_address_pool_ids = [azurerm_lb_backend_address_pool.lb_backend_pool.id]
      }
    }
  }
  lifecycle {
    ignore_changes = [instances, custom_data, tags]
  }
  depends_on = [
    azurerm_lb_backend_address_pool.lb_backend_pool, azurerm_lb_probe.backend_lb_probe,
    azurerm_proximity_placement_group.ppg, azurerm_lb_rule.backend_lb_rule, azurerm_lb_rule.ui_lb_rule
  ]
}


resource "null_resource" "force-delete-vmss" {
  triggers = {
    vmss_name       = azurerm_linux_virtual_machine_scale_set.vmss.name
    rg_name         = data.azurerm_resource_group.rg.name
    subscription_id = var.subscription_id
  }
  provisioner "local-exec" {
    when    = destroy
    command = "az vmss delete --name ${self.triggers.vmss_name} --resource-group ${self.triggers.rg_name} --force-deletion true --subscription ${self.triggers.subscription_id}"
  }
  depends_on = [azurerm_linux_virtual_machine_scale_set.vmss]
}
