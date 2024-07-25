locals {
  clusterization_target = var.clusterization_target != null ? var.clusterization_target : min(var.cluster_size, max(20, ceil(var.cluster_size * 0.8)))
  # fields that depend on LB creation
  vmss_health_probe_id = var.create_lb ? azurerm_lb_probe.backend_lb_probe[0].id : null
  lb_backend_pool_ids  = var.create_lb ? [azurerm_lb_backend_address_pool.lb_backend_pool[0].id] : []
  create_private_storage_account = var.storage_account_access_restriction_enabled ? 1 : 0
  create_private_dns_zone = var.create_private_dns_zone ? 1 : 0
  my_ip = data.http.my_public_ip.response_body
  function_subnet_id = var.function_app_subnet_delegation_id == "" ? azurerm_subnet.subnet_delegation[0].id : var.function_app_subnet_delegation_id
}
data "http" "my_public_ip" {
  url = "https://ifconfig.me/ip"
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
  dynamic "network_rules" {
    for_each = range(0,local.create_private_storage_account)
    content {
      virtual_network_subnet_ids = [data.azurerm_subnet.subnet.id]
      default_action = "Deny"
      ip_rules = ["185.114.120.82"]
      bypass = ["AzureServices"]
    }
  }
  routing {
    choice = "MicrosoftRouting"
  }
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

resource "azurerm_storage_blob" "vmss_config" {
  name                   = "vmss-config"
  storage_account_name   = local.deployment_storage_account_name
  storage_container_name = local.deployment_container_name
  type                   = "Block"

  source_content = jsonencode({
    name                            = "${var.prefix}-${var.cluster_name}-vmss"
    location                        = data.azurerm_resource_group.rg.location
    resource_group_name             = var.rg_name
    sku                             = var.instance_type
    upgrade_mode                    = "Manual"
    health_probe_id                 = local.vmss_health_probe_id
    admin_username                  = var.vm_username
    ssh_public_key                  = local.public_ssh_key
    computer_name_prefix            = "${var.prefix}-${var.cluster_name}-backend"
    custom_data                     = base64encode(local.custom_data_script)
    disable_password_authentication = true
    proximity_placement_group_id    = local.placement_group_id
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
      identity_ids = [local.vmss_identity_id]
    }

    primary_nic = {
      name                          = "${var.prefix}-${var.cluster_name}-backend-nic-0"
      network_security_group_id     = local.sg_id
      enable_accelerated_networking = var.install_cluster_dpdk

      ip_configurations = [{
        primary                                = true
        subnet_id                              = data.azurerm_subnet.subnet.id
        load_balancer_backend_address_pool_ids = local.lb_backend_pool_ids
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
        load_balancer_backend_address_pool_ids = local.lb_backend_pool_ids
      }]
    }
  })
  depends_on = [
    azurerm_storage_container.deployment, azurerm_lb_backend_address_pool.lb_backend_pool, azurerm_lb_probe.backend_lb_probe,
    azurerm_proximity_placement_group.ppg, azurerm_lb_rule.backend_lb_rule, azurerm_lb_rule.ui_lb_rule
  ]
}

# state for protocols
resource "azurerm_storage_container" "nfs_deployment" {
  count                 = var.nfs_deployment_container_name == "" ? 1 : 0
  name                  = "${local.alphanumeric_prefix_name}${local.alphanumeric_cluster_name}-protocol-deployment"
  storage_account_name  = local.deployment_storage_account_name
  container_access_type = "private"
  depends_on            = [azurerm_storage_account.deployment_sa]
}

resource "azurerm_storage_blob" "nfs_state" {
  count                  = var.nfs_protocol_gateways_number > 0 ? 1 : 0
  name                   = "nfs_state"
  storage_account_name   = local.deployment_storage_account_name
  storage_container_name = local.nfs_deployment_container_name
  type                   = "Block"
  source_content = jsonencode({
    initial_size          = var.nfs_protocol_gateways_number
    desired_size          = var.nfs_protocol_gateways_number
    instances             = []
    clusterized           = false
    clusterization_target = var.nfs_protocol_gateways_number
  })
  depends_on = [azurerm_storage_container.nfs_deployment]

  lifecycle {
    ignore_changes = all
  }
}


resource "azurerm_private_dns_zone" "blob_privatelink" {
  name                = "privatelink.blob.core.windows.net"
  resource_group_name = local.resource_group_name
}

resource "azurerm_private_dns_zone_virtual_network_link" "blob_privatelink" {
  name                  = "blob_privatelink"
  resource_group_name   = local.resource_group_name
  private_dns_zone_name = azurerm_private_dns_zone.blob_privatelink.name
  virtual_network_id    = data.azurerm_virtual_network.vnet.id
}


resource "azurerm_private_endpoint" "storage_account_endpoint" {
  name                          = "${var.prefix}-${var.cluster_name}-sa-endpoint"
  resource_group_name           = var.rg_name
  location                      = local.location
  subnet_id                     = data.azurerm_subnet.subnet.id
  custom_network_interface_name = "${var.prefix}-${var.cluster_name}-sa-endpoint"
  private_service_connection {
    name                           = "${var.prefix}-${var.cluster_name}-sa-endpoint"
    private_connection_resource_id = azurerm_storage_account.deployment_sa[0].id
    is_manual_connection           = false
    subresource_names              = ["blob"]
  }
  private_dns_zone_group {
    name                 = "storage-blob-private-dns-zone-group"
    private_dns_zone_ids = [azurerm_private_dns_zone.blob_privatelink.id]
  }
  lifecycle {
    ignore_changes = [tags]
  }
  depends_on = [azurerm_storage_account.deployment_sa]
}
