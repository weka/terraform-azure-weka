
locals {
  ssh_path                  = "/tmp/${var.prefix}-${var.cluster_name}"
  ssh_public_key_path       = "${local.ssh_path}-public-key.pub"
  ssh_private_key_path      = "${local.ssh_path}-private-key.pem"
  public_ssh_key            = var.ssh_public_key == null ? tls_private_key.ssh_key[0].public_key_openssh : var.ssh_public_key
  disk_size                 = var.default_disk_size + var.traces_per_ionode * (var.containers_config_map[var.instance_type].compute + var.containers_config_map[var.instance_type].drive + var.containers_config_map[var.instance_type].frontend)
  private_nic_first_index   = var.assign_public_ip ? 1 : 0
  alphanumeric_cluster_name = lower(replace(var.cluster_name, "/\\W|_|\\s/", ""))
  alphanumeric_prefix_name  = lower(replace(var.prefix, "/\\W|_|\\s/", ""))
  subnet_range              = data.azurerm_subnet.subnet.address_prefix
  nics_numbers              = var.install_cluster_dpdk ? var.containers_config_map[var.instance_type].nics : 1
  custom_data_script = templatefile("${path.module}/user-data.sh", {
    apt_repo_server          = var.apt_repo_server
    user                     = var.vm_username
    install_cluster_dpdk     = var.install_cluster_dpdk
    subnet_range             = local.subnet_range
    nics_num                 = local.nics_numbers
    deploy_url               = "https://${azurerm_linux_function_app.function_app.name}.azurewebsites.net/api/deploy"
    report_url               = "https://${azurerm_linux_function_app.function_app.name}.azurewebsites.net/api/report"
    function_app_default_key = data.azurerm_function_app_host_keys.function_keys.default_function_key
    disk_size                = local.disk_size
  })
  placement_group_id = var.placement_group_id != "" ? var.placement_group_id : azurerm_proximity_placement_group.ppg[0].id
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
  filename        = local.ssh_public_key_path
  file_permission = "0600"
}

resource "local_file" "private_key" {
  count           = var.ssh_public_key == null ? 1 : 0
  content         = tls_private_key.ssh_key[count.index].private_key_pem
  filename        = local.ssh_private_key_path
  file_permission = "0600"
}

resource "azurerm_proximity_placement_group" "ppg" {
  count               = var.placement_group_id == "" ? 1 : 0
  name                = "${var.prefix}-${var.cluster_name}-backend-ppg"
  location            = data.azurerm_resource_group.rg.location
  zone                = var.zone
  allowed_vm_sizes    = [var.instance_type]
  resource_group_name = var.rg_name
  tags                = merge(var.tags_map, { "weka_cluster" : var.cluster_name })
  lifecycle {
    ignore_changes = [tags]
  }
}

resource "azurerm_linux_virtual_machine_scale_set" "vmss" {
  name                            = "${var.prefix}-${var.cluster_name}-vmss"
  location                        = data.azurerm_resource_group.rg.location
  zones                           = [var.zone]
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
  single_placement_group          = var.vmss_single_placement_group
  source_image_id                 = var.source_image_id
  overprovision                   = false
  tags = merge(var.tags_map, {
    "weka_cluster" : var.cluster_name, "user_id" : data.azurerm_client_config.current.object_id
  })

  os_disk {
    caching              = "ReadWrite"
    storage_account_type = "Premium_LRS"
  }
  data_disk {
    lun                  = 0
    caching              = "None"
    create_option        = "Empty"
    disk_size_gb         = local.disk_size
    storage_account_type = "Premium_LRS"
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
      network_security_group_id     = local.sg_id
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
      network_security_group_id     = local.sg_id
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
      network_security_group_id     = local.sg_id
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
    azurerm_proximity_placement_group.ppg, azurerm_lb_rule.backend_lb_rule, azurerm_lb_rule.ui_lb_rule,
  ]
}


resource "azurerm_role_assignment" "storage_blob_data_reader" {
  count                = var.weka_tar_storage_account_id != "" ? 1 : 0
  scope                = var.weka_tar_storage_account_id
  role_definition_name = "Storage Blob Data Reader"
  principal_id         = azurerm_linux_virtual_machine_scale_set.vmss.identity[0].principal_id
  depends_on           = [azurerm_linux_virtual_machine_scale_set.vmss]
}
