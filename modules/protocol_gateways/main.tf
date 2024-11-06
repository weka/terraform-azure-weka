data "azurerm_client_config" "current" {}

data "azurerm_resource_group" "rg" {
  name = var.rg_name
}

data "azurerm_subnet" "subnet" {
  resource_group_name  = var.vnet_rg_name
  virtual_network_name = var.vnet_name
  name                 = var.subnet_name
}

resource "azurerm_public_ip" "this" {
  count               = var.assign_public_ip && var.protocol != "NFS" ? var.gateways_number : 0
  name                = "${var.gateways_name}-public-ip-${count.index}"
  location            = var.location
  resource_group_name = var.rg_name
  allocation_method   = "Dynamic"
  sku                 = "Basic"
}

resource "azurerm_network_interface" "primary_gateway_nic_public" {
  count                          = var.assign_public_ip && var.protocol != "NFS" ? var.gateways_number : 0
  name                           = "${var.gateways_name}-primary-nic-${count.index}"
  location                       = var.location
  resource_group_name            = var.rg_name
  accelerated_networking_enabled = true

  ip_configuration {
    primary                       = true
    name                          = "ipconfig0"
    subnet_id                     = data.azurerm_subnet.subnet.id
    private_ip_address_allocation = "Dynamic"
    public_ip_address_id          = azurerm_public_ip.this[count.index].id
  }

  # secondary ips (floating ip)
  dynamic "ip_configuration" {
    for_each = range(var.secondary_ips_per_nic)
    content {
      name                          = "ipconfig${ip_configuration.value + 1}"
      subnet_id                     = data.azurerm_subnet.subnet.id
      private_ip_address_allocation = "Dynamic"
    }
  }
}

resource "azurerm_network_interface_security_group_association" "primary_gateway_nic_public" {
  count                     = var.assign_public_ip && var.protocol != "NFS" ? var.gateways_number : 0
  network_interface_id      = azurerm_network_interface.primary_gateway_nic_public[count.index].id
  network_security_group_id = var.sg_id
}

resource "azurerm_network_interface" "primary_gateway_nic_private" {
  count                          = var.assign_public_ip || var.protocol == "NFS" ? 0 : var.gateways_number
  name                           = "${var.gateways_name}-primary-nic-${count.index}"
  location                       = var.location
  resource_group_name            = var.rg_name
  accelerated_networking_enabled = true

  ip_configuration {
    primary                       = true
    name                          = "ipconfig0"
    subnet_id                     = data.azurerm_subnet.subnet.id
    private_ip_address_allocation = "Dynamic"
  }

  # secondary ips (floating ip)
  dynamic "ip_configuration" {
    for_each = range(var.secondary_ips_per_nic)
    content {
      name                          = "ipconfig${ip_configuration.value + 1}"
      subnet_id                     = data.azurerm_subnet.subnet.id
      private_ip_address_allocation = "Dynamic"
    }
  }
}

resource "azurerm_network_interface_security_group_association" "primary_gateway_nic_private" {
  count                     = var.assign_public_ip || var.protocol == "NFS" ? 0 : var.gateways_number
  network_interface_id      = azurerm_network_interface.primary_gateway_nic_private[count.index].id
  network_security_group_id = var.sg_id
}

locals {
  secondary_nics_num = (local.nics_numbers - 1) * var.gateways_number
}

resource "azurerm_network_interface" "secondary_gateway_nic" {
  count                          = var.protocol != "NFS" ? local.secondary_nics_num : 0
  name                           = "${var.gateways_name}-secondary-nic-${count.index + var.gateways_number}"
  location                       = var.location
  resource_group_name            = var.rg_name
  accelerated_networking_enabled = true

  ip_configuration {
    primary                       = true
    name                          = "ipconfig0"
    subnet_id                     = data.azurerm_subnet.subnet.id
    private_ip_address_allocation = "Dynamic"
  }
}

resource "azurerm_network_interface_security_group_association" "secondary_gateway_nic" {
  count                     = var.protocol != "NFS" ? local.secondary_nics_num : 0
  network_interface_id      = azurerm_network_interface.secondary_gateway_nic[count.index].id
  network_security_group_id = var.sg_id
}

locals {
  disk_size             = var.disk_size + var.traces_per_frontend * var.frontend_container_cores_num
  first_nic_ids         = var.assign_public_ip ? azurerm_network_interface.primary_gateway_nic_public[*].id : azurerm_network_interface.primary_gateway_nic_private[*].id
  first_nic_private_ips = var.assign_public_ip ? azurerm_network_interface.primary_gateway_nic_public[*].private_ip_address : azurerm_network_interface.primary_gateway_nic_private[*].private_ip_address
  nics_numbers          = var.frontend_container_cores_num + 1

  init_script = templatefile("${path.module}/init.sh", {
    apt_repo_server          = var.apt_repo_server
    nics_num                 = local.nics_numbers
    subnet_range             = data.azurerm_subnet.subnet.address_prefixes[0]
    disk_size                = local.disk_size
    deploy_url               = var.deploy_function_url
    report_url               = var.report_function_url
    function_app_default_key = var.function_app_default_key
    protocol                 = lower(var.protocol)
  })

  setup_smb_protocol_script = templatefile("${path.module}/setup_smb.sh", {
    cluster_name                 = var.smb_cluster_name
    domain_name                  = var.smb_domain_name
    smbw_enabled                 = var.smbw_enabled
    gateways_number              = var.gateways_number
    gateways_name                = var.gateways_name
    frontend_container_cores_num = var.frontend_container_cores_num
    report_function_url          = var.report_function_url
    vault_function_app_key_name  = var.vault_function_app_key_name
    key_vault_url                = var.key_vault_url
  })

  setup_s3_protocol_script = file("${path.module}/setup_s3.sh")

  setup_validation_script = templatefile("${path.module}/setup_validation.sh", {
    gateways_number             = var.gateways_number
    gateways_name               = var.gateways_name
    report_function_url         = var.report_function_url
    vault_function_app_key_name = var.vault_function_app_key_name
    key_vault_url               = var.key_vault_url
    protocol                    = var.protocol
    smbw_enabled                = var.smbw_enabled
  })

  smb_protocol_script = var.protocol == "SMB" ? local.setup_smb_protocol_script : ""
  s3_protocol_script  = var.protocol == "S3" ? local.setup_s3_protocol_script : ""
  validation_script   = var.setup_protocol && (var.protocol == "SMB" || var.protocol == "S3") ? local.setup_validation_script : ""

  setup_protocol_script = var.setup_protocol ? compact([local.smb_protocol_script, local.s3_protocol_script]) : []

  custom_data_parts = concat([local.init_script, local.validation_script], local.setup_protocol_script)

  custom_data = join("\n", local.custom_data_parts)

  gw_identity_id        = var.vm_identity_name == "" ? azurerm_user_assigned_identity.this[0].id : data.azurerm_user_assigned_identity.this[0].id
  gw_identity_principal = var.vm_identity_name == "" ? azurerm_user_assigned_identity.this[0].principal_id : data.azurerm_user_assigned_identity.this[0].principal_id
}


resource "azurerm_linux_virtual_machine" "this" {
  count                           = var.protocol != "NFS" ? var.gateways_number : 0
  name                            = "${var.gateways_name}-vm-${count.index}"
  computer_name                   = "${var.gateways_name}-${count.index}"
  location                        = var.location
  resource_group_name             = var.rg_name
  size                            = var.instance_type
  admin_username                  = var.vm_username
  custom_data                     = base64encode(local.custom_data)
  proximity_placement_group_id    = var.ppg_id
  disable_password_authentication = true
  source_image_id                 = var.source_image_id
  tags                            = merge(var.tags_map, { "weka_protocol_gateways" : var.gateways_name, "user_id" : data.azurerm_client_config.current.object_id })

  network_interface_ids = concat(
    [local.first_nic_ids[count.index]],
    slice(azurerm_network_interface.secondary_gateway_nic[*].id, (local.nics_numbers - 1) * count.index, (local.nics_numbers - 1) * (count.index + 1))
  )

  os_disk {
    caching              = "ReadWrite"
    name                 = "os-disk-${var.gateways_name}-${count.index}"
    storage_account_type = "StandardSSD_LRS"
  }

  admin_ssh_key {
    username   = var.vm_username
    public_key = var.ssh_public_key
  }

  identity {
    type         = "UserAssigned"
    identity_ids = [local.gw_identity_id]
  }

  lifecycle {
    ignore_changes = [tags, custom_data]
    precondition {
      condition     = var.protocol == "S3" ? var.gateways_number >= 1 : var.gateways_number >= 3 && var.gateways_number <= 8
      error_message = "The amount of protocol gateways should be at least 1 for S3 and at least 3 and at most 8 for SMB."
    }
    precondition {
      condition     = var.protocol == "SMB" && var.setup_protocol ? var.smb_domain_name != "" : true
      error_message = "The SMB domain name should be set when deploying SMB protocol gateways."
    }
    precondition {
      condition     = var.secondary_ips_per_nic <= 3
      error_message = "The number of secondary IPs per single NIC per protocol gateway virtual machine must be at most 3 for SMB."
    }
    precondition {
      condition     = var.frontend_container_cores_num < local.nics_numbers
      error_message = "The number of frontends must be less than the number of NICs."
    }
    precondition {
      condition     = var.location == data.azurerm_resource_group.rg.location
      error_message = "The location of the protocol gateways must be the same as the location of the resource group."
    }
  }
  depends_on = [azurerm_network_interface.primary_gateway_nic_private, azurerm_network_interface.primary_gateway_nic_public, azurerm_network_interface.secondary_gateway_nic]
}

resource "azurerm_managed_disk" "this" {
  count                = var.protocol != "NFS" ? var.gateways_number : 0
  name                 = "weka-disk-${var.gateways_name}-${count.index}"
  location             = var.location
  resource_group_name  = var.rg_name
  storage_account_type = "StandardSSD_LRS"
  create_option        = "Empty"
  disk_size_gb         = local.disk_size
}

resource "azurerm_virtual_machine_data_disk_attachment" "this" {
  count              = var.protocol != "NFS" ? var.gateways_number : 0
  managed_disk_id    = azurerm_managed_disk.this[count.index].id
  virtual_machine_id = azurerm_linux_virtual_machine.this[count.index].id
  lun                = 0
  caching            = "ReadWrite"
  depends_on         = [azurerm_linux_virtual_machine.this]
}

data "azurerm_user_assigned_identity" "this" {
  count               = var.vm_identity_name != "" ? 1 : 0
  name                = var.vm_identity_name
  resource_group_name = data.azurerm_resource_group.rg.name
}

resource "azurerm_user_assigned_identity" "this" {
  count               = var.vm_identity_name == "" ? 1 : 0
  location            = data.azurerm_resource_group.rg.location
  name                = "${var.gateways_name}-identity"
  resource_group_name = data.azurerm_resource_group.rg.name
}

resource "azurerm_key_vault_access_policy" "gateways_vmss_key_vault" {
  key_vault_id = var.key_vault_id
  tenant_id    = data.azurerm_client_config.current.tenant_id
  object_id    = local.gw_identity_principal
  secret_permissions = [
    "Get",
  ]
}

resource "azurerm_role_assignment" "gateways_vm_key_vault" {
  count                = var.vm_identity_name == "" ? 1 : 0
  scope                = var.key_vault_id
  role_definition_name = "Key Vault Secrets User"
  principal_id         = azurerm_user_assigned_identity.this[0].principal_id
}

resource "azurerm_role_assignment" "weka_tar_data_reader" {
  count                = var.vm_identity_name == "" && var.weka_tar_storage_account_id != "" ? 1 : 0
  scope                = var.weka_tar_storage_account_id
  role_definition_name = "Storage Blob Data Reader"
  principal_id         = azurerm_user_assigned_identity.this[0].principal_id
}

resource "azurerm_role_assignment" "reader" {
  count                = var.vm_identity_name == "" ? 1 : 0
  scope                = data.azurerm_resource_group.rg.id
  role_definition_name = "Reader"
  principal_id         = azurerm_user_assigned_identity.this[0].principal_id
}

# needed for floating-ip support
resource "azurerm_role_assignment" "network_contributor" {
  count                = var.vm_identity_name == "" && var.protocol == "NFS" ? 1 : 0
  scope                = data.azurerm_resource_group.rg.id
  role_definition_name = "Network Contributor"
  principal_id         = azurerm_user_assigned_identity.this[0].principal_id
}

resource "azurerm_orchestrated_virtual_machine_scale_set" "nfs" {
  count                        = var.protocol == "NFS" ? 1 : 0
  name                         = "${var.gateways_name}-vmss"
  location                     = var.location
  resource_group_name          = var.rg_name
  instances                    = 0 # will be set to var.gateways_number by scale_up workflow
  platform_fault_domain_count  = 1
  sku_name                     = var.instance_type
  tags                         = merge(var.tags_map, { "weka_protocol_gateways" : var.gateways_name, "user_id" : data.azurerm_client_config.current.object_id })
  proximity_placement_group_id = var.ppg_id
  source_image_id              = var.source_image_id

  os_profile {
    custom_data = base64encode(local.custom_data)
    linux_configuration {
      computer_name_prefix            = var.gateways_name
      disable_password_authentication = true
      admin_username                  = var.vm_username

      admin_ssh_key {
        username   = var.vm_username
        public_key = var.ssh_public_key
      }
    }
  }

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

  identity {
    type         = "UserAssigned"
    identity_ids = [local.gw_identity_id]
  }

  network_interface {
    name                          = "${var.gateways_name}-primary-nic-0"
    network_security_group_id     = var.sg_id
    primary                       = true
    enable_accelerated_networking = true

    # ipconfig with public ip
    dynamic "ip_configuration" {
      for_each = range(var.assign_public_ip ? 1 : 0)
      content {
        primary   = true
        name      = "ipconfig0"
        subnet_id = data.azurerm_subnet.subnet.id
        public_ip_address {
          name              = "${var.gateways_name}-public-ip"
          domain_name_label = "nfs-gw-public-ip"
        }
      }
    }

    # ipconfig without public ip
    dynamic "ip_configuration" {
      for_each = range(var.assign_public_ip ? 0 : 1)
      content {
        primary   = true
        name      = "ipconfig0"
        subnet_id = data.azurerm_subnet.subnet.id
      }
    }

    # secondary ips (floating ip)
    dynamic "ip_configuration" {
      for_each = range(var.secondary_ips_per_nic)
      content {
        name      = "ipconfig${ip_configuration.value + 1}"
        subnet_id = data.azurerm_subnet.subnet.id
      }
    }
  }


  dynamic "network_interface" {
    for_each = range(1, local.nics_numbers)
    content {
      name                          = "${var.gateways_name}-secondary-nic-${network_interface.value}"
      network_security_group_id     = var.sg_id
      primary                       = false
      enable_accelerated_networking = true
      ip_configuration {
        primary   = true
        name      = "ipconfig0"
        subnet_id = data.azurerm_subnet.subnet.id
      }
    }
  }
  lifecycle {
    ignore_changes = [tags, os_profile[0].custom_data, instances]
    precondition {
      condition     = var.gateways_number >= 1
      error_message = "The amount of protocol gateways should be at least 1 for NFS."
    }
    precondition {
      condition     = var.frontend_container_cores_num < local.nics_numbers
      error_message = "The number of frontends must be less than the number of NICs."
    }
    precondition {
      condition     = var.location == data.azurerm_resource_group.rg.location
      error_message = "The location of the protocol gateways must be the same as the location of the resource group."
    }
  }
}
