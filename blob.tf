locals {
  clusterization_target = var.clusterization_target != null ? var.clusterization_target : min(var.cluster_size, max(20, ceil(var.cluster_size * 0.8)))
}


resource "azurerm_storage_account" "deployment_sa" {
  count                    = var.deployment_storage_account_name == "" ? 1 : 0
  name                     = substr("${local.alphanumeric_prefix_name}${local.alphanumeric_cluster_name}deployment", 0, 24)
  location                 = data.azurerm_resource_group.rg.location
  resource_group_name      = var.rg_name
  account_kind             = "StorageV2"
  account_tier             = "Standard"
  account_replication_type = "ZRS"
  tags                     = merge(var.tags_map, { "weka_cluster" : var.cluster_name })
  lifecycle {
    ignore_changes = [tags]
  }
}

resource "azurerm_storage_container" "deployment" {
  count                 = var.deployment_container_name == "" ? 1 : 0
  name                  = "${local.alphanumeric_prefix_name}${local.alphanumeric_cluster_name}-deployment"
  storage_account_name  = local.deployment_storage_account_name
  container_access_type = "private"
  depends_on            = [azurerm_storage_account.deployment_sa]
}

resource "azurerm_storage_blob" "state" {
  name                   = "state"
  storage_account_name   = local.deployment_storage_account_name
  storage_container_name = local.deployment_container_name
  type                   = "Block"
  source_content         = "{\"initial_size\":${var.cluster_size}, \"desired_size\":${var.cluster_size}, \"instances\":[], \"clusterized\":false, \"clusterization_target\":${local.clusterization_target}}"
  depends_on             = [azurerm_storage_container.deployment]

  lifecycle {
    ignore_changes = all
  }
}

data "azurerm_storage_account" "deployment_blob" {
  count               = var.deployment_storage_account_name != "" ? 1 : 0
  name                = var.deployment_storage_account_name
  resource_group_name = var.rg_name
}

data "azurerm_storage_account" "obs_sa" {
  count               = var.tiering_obs_name != "" ? 1 : 0
  name                = var.tiering_obs_name
  resource_group_name = var.rg_name
}

resource "azurerm_storage_blob" "vmss_config" {
  name                   = "vmss-config"
  storage_account_name   = local.deployment_storage_account_name
  storage_container_name = local.deployment_container_name
  type                   = "Block"

  source_content = jsonencode({
    name                            = "${var.prefix}-${var.cluster_name}-vmss"
    location                        = data.azurerm_resource_group.rg.location
    zones                           = [var.zone]
    resource_group_name             = var.rg_name
    sku                             = var.instance_type
    upgrade_mode                    = "Manual"
    health_probe_id                 = azurerm_lb_probe.backend_lb_probe.id
    admin_username                  = var.vm_username
    ssh_public_key                  = local.public_ssh_key
    computer_name_prefix            = "${var.prefix}-${var.cluster_name}-backend"
    custom_data                     = base64encode(local.custom_data_script)
    disable_password_authentication = true
    proximity_placement_group_id    = var.vmss_single_placement_group ? local.placement_group_id : null
    single_placement_group          = var.vmss_single_placement_group
    source_image_id                 = var.source_image_id
    overprovision                   = false
    orchestration_mode              = "Uniform"
    tags = merge(var.tags_map, {
      "weka_cluster" : var.cluster_name,
      "user_id" : data.azurerm_client_config.current.object_id,
    })

    os_disk = {
      caching              = "ReadWrite"
      storage_account_type = "Premium_LRS"
    }

    data_disk = {
      lun                  = 0
      caching              = "None"
      create_option        = "Empty"
      disk_size_gb         = local.disk_size
      storage_account_type = "Premium_LRS"
    }

    identity = {
      type         = "UserAssigned"
      identity_ids = [azurerm_user_assigned_identity.vmss.id]
    }

    primary_nic = {
      name                          = "${var.prefix}-${var.cluster_name}-backend-nic-0"
      network_security_group_id     = local.sg_id
      enable_accelerated_networking = var.install_cluster_dpdk

      ip_configurations = [{
        primary                                = true
        subnet_id                              = data.azurerm_subnet.subnet.id
        load_balancer_backend_address_pool_ids = [azurerm_lb_backend_address_pool.lb_backend_pool.id]
        public_ip_address = {
          assign            = local.assign_public_ip
          name              = "${var.prefix}-${var.cluster_name}-public-ip"
          domain_name_label = "${var.prefix}-${var.cluster_name}-backend"
        }
      }]
    }

    secondary_nics = {
      number                        = local.nics_numbers - 1
      name_prefix                   = "${var.prefix}-${var.cluster_name}-backend-nic"
      network_security_group_id     = local.sg_id
      enable_accelerated_networking = var.install_cluster_dpdk
      ip_configurations = [{
        primary                                = true
        subnet_id                              = data.azurerm_subnet.subnet.id
        load_balancer_backend_address_pool_ids = [azurerm_lb_backend_address_pool.lb_backend_pool.id]
      }]
    }
  })
  depends_on = [
    azurerm_storage_container.deployment, azurerm_lb_backend_address_pool.lb_backend_pool, azurerm_lb_probe.backend_lb_probe,
    azurerm_proximity_placement_group.ppg, azurerm_lb_rule.backend_lb_rule, azurerm_lb_rule.ui_lb_rule
  ]
}
