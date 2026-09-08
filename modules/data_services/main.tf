data "azurerm_client_config" "current" {}

data "azurerm_resource_group" "rg" {
  name = var.rg_name
}

data "azurerm_subnet" "subnet" {
  resource_group_name  = var.vnet_rg_name
  virtual_network_name = var.vnet_name
  name                 = var.subnet_name
}

locals {
  use_marketplace_image = var.image_sku != null && var.image_offer != null
  vmss_name             = "${var.data_services_name}-vmss"

  init_script = templatefile("${path.module}/init.sh", {
    apt_repo_server          = var.apt_repo_server
    subnet_range             = data.azurerm_subnet.subnet.address_prefixes[0]
    disk_size                = var.disk_size
    deploy_url               = var.deploy_function_url
    report_url               = var.report_function_url
    function_app_default_key = var.function_app_default_key
  })

  identity_id        = var.vm_identity_name == "" ? azurerm_user_assigned_identity.this[0].id : data.azurerm_user_assigned_identity.this[0].id
  identity_principal = var.vm_identity_name == "" ? azurerm_user_assigned_identity.this[0].principal_id : data.azurerm_user_assigned_identity.this[0].principal_id
}

resource "azurerm_linux_virtual_machine_scale_set" "this" {
  name                            = local.vmss_name
  computer_name_prefix            = var.data_services_name
  location                        = var.location
  resource_group_name             = var.rg_name
  sku                             = var.instance_type
  instances                       = var.data_services_number
  admin_username                  = var.vm_username
  custom_data                     = base64encode(local.init_script)
  disable_password_authentication = true
  source_image_id                 = local.use_marketplace_image ? "" : var.source_image_id
  upgrade_mode                    = "Manual"
  overprovision                   = false
  tags                            = merge(var.tags_map, { "weka_data_services" : var.data_services_name, "weka_cluster" : var.cluster_name, "user_id" : data.azurerm_client_config.current.object_id })

  dynamic "source_image_reference" {
    for_each = local.use_marketplace_image ? [1] : []
    content {
      publisher = "Canonical"
      offer     = var.image_offer
      sku       = var.image_sku
      version   = var.image_version
    }
  }

  admin_ssh_key {
    username   = var.vm_username
    public_key = var.ssh_public_key
  }

  identity {
    type         = "UserAssigned"
    identity_ids = [local.identity_id]
  }

  network_interface {
    name                          = "${var.data_services_name}-nic-0"
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
          name = "${var.data_services_name}-public-ip"
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
  }

  os_disk {
    caching              = "ReadWrite"
    storage_account_type = "StandardSSD_LRS"
    disk_size_gb         = var.root_volume_size
  }

  data_disk {
    lun                  = 0
    caching              = "ReadWrite"
    create_option        = "Empty"
    disk_size_gb         = var.disk_size
    storage_account_type = "StandardSSD_LRS"
  }

  lifecycle {
    ignore_changes = [tags, custom_data, source_image_id]
    precondition {
      condition     = var.location == data.azurerm_resource_group.rg.location
      error_message = "The location of the data services instances must be the same as the location of the resource group."
    }
  }
}

data "azurerm_user_assigned_identity" "this" {
  count               = var.vm_identity_name != "" ? 1 : 0
  name                = var.vm_identity_name
  resource_group_name = data.azurerm_resource_group.rg.name
}

resource "azurerm_user_assigned_identity" "this" {
  count               = var.vm_identity_name == "" ? 1 : 0
  location            = data.azurerm_resource_group.rg.location
  name                = "${var.data_services_name}-identity"
  resource_group_name = data.azurerm_resource_group.rg.name
}

resource "azurerm_key_vault_access_policy" "data_services_key_vault" {
  key_vault_id = var.key_vault_id
  tenant_id    = data.azurerm_client_config.current.tenant_id
  object_id    = local.identity_principal
  secret_permissions = [
    "Get",
  ]
}

resource "azurerm_role_assignment" "data_services_key_vault" {
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
