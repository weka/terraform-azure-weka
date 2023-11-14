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
  count               = var.assign_public_ip ? var.gateways_number : 0
  name                = "${var.gateways_name}-public-ip-${count.index}"
  location            = data.azurerm_resource_group.rg.location
  resource_group_name = var.rg_name
  allocation_method   = "Dynamic"
}

resource "azurerm_network_interface" "primary_gateway_nic_public" {
  count                         = var.assign_public_ip ? var.gateways_number : 0
  name                          = "${var.gateways_name}-primary-nic-${count.index}"
  location                      = data.azurerm_resource_group.rg.location
  resource_group_name           = var.rg_name
  enable_accelerated_networking = true

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
  count                     = var.assign_public_ip ? var.gateways_number : 0
  network_interface_id      = azurerm_network_interface.primary_gateway_nic_public[count.index].id
  network_security_group_id = var.sg_id
}

resource "azurerm_network_interface" "primary_gateway_nic_private" {
  count                         = var.assign_public_ip ? 0 : var.gateways_number
  name                          = "${var.gateways_name}-primary-nic-${count.index}"
  location                      = data.azurerm_resource_group.rg.location
  resource_group_name           = var.rg_name
  enable_accelerated_networking = true

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
  count                     = var.assign_public_ip ? 0 : var.gateways_number
  network_interface_id      = azurerm_network_interface.primary_gateway_nic_private[count.index].id
  network_security_group_id = var.sg_id
}

locals {
  secondary_nics_num = (local.nics_numbers - 1) * var.gateways_number
}

resource "azurerm_network_interface" "secondary_gateway_nic" {
  count                         = local.secondary_nics_num
  name                          = "${var.gateways_name}-secondary-nic-${count.index + var.gateways_number}"
  location                      = data.azurerm_resource_group.rg.location
  resource_group_name           = var.rg_name
  enable_accelerated_networking = true

  ip_configuration {
    primary                       = true
    name                          = "ipconfig0"
    subnet_id                     = data.azurerm_subnet.subnet.id
    private_ip_address_allocation = "Dynamic"
  }
}

resource "azurerm_network_interface_security_group_association" "secondary_gateway_nic" {
  count                     = local.secondary_nics_num
  network_interface_id      = azurerm_network_interface.secondary_gateway_nic[count.index].id
  network_security_group_id = var.sg_id
}

locals {
  disk_size             = var.disk_size + var.traces_per_frontend * var.frontend_cores_num
  first_nic_ids         = var.assign_public_ip ? azurerm_network_interface.primary_gateway_nic_public[*].id : azurerm_network_interface.primary_gateway_nic_private[*].id
  first_nic_private_ips = var.assign_public_ip ? azurerm_network_interface.primary_gateway_nic_public[*].private_ip_address : azurerm_network_interface.primary_gateway_nic_private[*].private_ip_address
  nics_numbers          = var.frontend_cores_num + 1
  init_script = templatefile("${path.module}/init.sh", {
    apt_repo_server  = var.apt_repo_server
    nics_num         = local.nics_numbers
    subnet_range     = data.azurerm_subnet.subnet.address_prefix
    disk_size        = local.disk_size
    install_weka_url = var.install_weka_url
    key_vault_url    = data.azurerm_key_vault.this.vault_uri
  })

  deploy_script = templatefile("${path.module}/deploy_protocol_gateways.sh", {
    frontend_cores_num = var.frontend_cores_num
    subnet_prefixes    = data.azurerm_subnet.subnet.address_prefix
    backend_lb_ip      = var.backend_lb_ip
    key_vault_url      = data.azurerm_key_vault.this.vault_uri
  })

  setup_nfs_protocol_script = templatefile("${path.module}/setup_nfs.sh", {
    gateways_name        = var.gateways_name
    interface_group_name = var.interface_group_name
    client_group_name    = var.client_group_name
  })

  setup_smb_protocol_script = templatefile("${path.module}/setup_smb.sh", {
    cluster_name        = var.smb_cluster_name
    domain_name         = var.smb_domain_name
    domain_netbios_name = var.smb_domain_netbios_name
    smbw_enabled        = var.smbw_enabled
    dns_ip              = var.smb_dns_ip_address
    gateways_number     = var.gateways_number
    gateways_name       = var.gateways_name
    frontend_cores_num  = var.frontend_cores_num
    share_name          = var.smb_share_name
  })

  protocol_script = var.protocol == "NFS" ? local.setup_nfs_protocol_script : local.setup_smb_protocol_script

  setup_protocol_script = var.setup_protocol ? local.protocol_script : ""

  custom_data_parts = [
    local.init_script, local.deploy_script, local.setup_protocol_script
  ]
  custom_data = join("\n", local.custom_data_parts)
}


resource "azurerm_linux_virtual_machine" "this" {
  count                           = var.gateways_number
  name                            = "${var.gateways_name}-vm-${count.index}"
  computer_name                   = "${var.gateways_name}-${count.index}"
  location                        = data.azurerm_resource_group.rg.location
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
    type = "SystemAssigned"
  }

  lifecycle {
    ignore_changes = [tags]
    precondition {
      condition     = var.protocol == "NFS" ? var.gateways_number >= 1 : var.gateways_number >= 3 && var.gateways_number <= 8
      error_message = "The amount of protocol gateways should be at least 1 for NFS and at least 3 and at most 8 for SMB."
    }
    precondition {
      condition     = var.protocol == "SMB" ? var.smb_domain_name != "" : true
      error_message = "The SMB domain name should be set when deploying SMB protocol gateways."
    }
    precondition {
      condition     = var.protocol == "SMB" ? var.secondary_ips_per_nic <= 3 : true
      error_message = "The number of secondary IPs per single NIC per protocol gateway virtual machine must be at most 3 for SMB."
    }
    precondition {
      condition     = var.frontend_cores_num < local.nics_numbers
      error_message = "The number of frontends must be less than the number of NICs."
    }
  }
  depends_on = [azurerm_network_interface.primary_gateway_nic_private, azurerm_network_interface.primary_gateway_nic_public, azurerm_network_interface.secondary_gateway_nic]
}

resource "azurerm_managed_disk" "this" {
  count                = var.gateways_number
  name                 = "weka-disk-${var.gateways_name}-${count.index}"
  location             = data.azurerm_resource_group.rg.location
  resource_group_name  = var.rg_name
  storage_account_type = "StandardSSD_LRS"
  create_option        = "Empty"
  disk_size_gb         = local.disk_size
}

resource "azurerm_virtual_machine_data_disk_attachment" "this" {
  count              = var.gateways_number
  managed_disk_id    = azurerm_managed_disk.this[count.index].id
  virtual_machine_id = azurerm_linux_virtual_machine.this[count.index].id
  lun                = 0
  caching            = "ReadWrite"
  depends_on         = [azurerm_linux_virtual_machine.this]
}

data "azurerm_key_vault" "this" {
  name                = var.key_vault_name
  resource_group_name = var.rg_name
}

resource "azurerm_key_vault_access_policy" "gateways_vmss_key_vault" {
  count        = var.gateways_number
  key_vault_id = data.azurerm_key_vault.this.id
  tenant_id    = data.azurerm_client_config.current.tenant_id
  object_id    = azurerm_linux_virtual_machine.this[count.index].identity[0].principal_id
  secret_permissions = [
    "Get",
  ]
  depends_on = [azurerm_linux_virtual_machine.this]
}

resource "azurerm_role_assignment" "gateways_vmss_key_vault" {
  count                = var.gateways_number
  scope                = data.azurerm_key_vault.this.id
  role_definition_name = "Key Vault Secrets User"
  principal_id         = azurerm_linux_virtual_machine.this[count.index].identity[0].principal_id
  depends_on           = [azurerm_linux_virtual_machine.this]
}

resource "azurerm_role_assignment" "storage_blob_data_reader" {
  count                = var.weka_tar_storage_account_id != "" ? var.gateways_number : 0
  scope                = var.weka_tar_storage_account_id
  role_definition_name = "Storage Blob Data Reader"
  principal_id         = azurerm_linux_virtual_machine.this[count.index].identity[0].principal_id
  depends_on           = [azurerm_linux_virtual_machine.this]
}
