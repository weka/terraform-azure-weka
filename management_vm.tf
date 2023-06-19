locals {
  function_code_path     = "${path.module}/app/code"
  function_app_code_hash = md5(join("", [for f in fileset(local.function_code_path, "**") : filemd5("${local.function_code_path}/${f}")]))
  function_bins_dir      = "${path.module}/.tf-binaries"
  os                     = "darwin"
}

locals {
  location              = data.azurerm_resource_group.rg.location
  function_bin_name = "${var.function_app_dist}/${var.app_code_hash}"
  weka_sa               = "${var.function_app_storage_account_prefix}${local.location}"
  weka_sa_container     = "${var.function_app_storage_account_container_prefix}${local.location}"
  code_url         = "https://${local.weka_sa}.blob.core.windows.net/${local.weka_sa_container}/${local.function_bin_name}"
}

locals {
  stripe_width_calculated = var.cluster_size - var.protection_level - 1
  stripe_width            = local.stripe_width_calculated < 16 ? local.stripe_width_calculated : 16
}

data "azurerm_subscription" "primary" {}

data "azuread_client_config" "current" {}

resource "azuread_application" "auth" {
  display_name = "${var.prefix}-${var.cluster_name}-auth"
  owners       = [data.azuread_client_config.current.object_id]
}

resource "azuread_service_principal" "sp" {
  application_id = azuread_application.auth.application_id
  app_role_assignment_required = false
  owners = [data.azuread_client_config.current.object_id]
}

resource "azuread_service_principal_password" "sp_password" {
  service_principal_id = azuread_service_principal.sp.object_id
}

data "template_file" "management-init" {
  template = file("${path.module}/management-vm-init.sh")
  vars = {
    azure_client_id         = azuread_service_principal.sp.application_id
    azure_client_secret     = azuread_service_principal_password.sp_password.value
    azure_tenant_id         = azuread_service_principal.sp.application_tenant_id
    function_app_code_url   = local.code_url
    http_server_port        = var.http_server_port
    state_storage_name      = azurerm_storage_account.deployment_sa.name
    state_container_name    = azurerm_storage_container.deployment.name
    hosts_num               = var.cluster_size
    cluster_name            = var.cluster_name
    protection_level        = var.protection_level
    stripe_width            = var.stripe_width != -1 ? var.stripe_width : local.stripe_width
    hotspare                = var.hotspare
    vm_username             = var.vm_username
    compute_memory          = var.container_number_map[var.instance_type].memory
    subscription_id         = data.azurerm_subscription.primary.subscription_id
    resource_group_name     = data.azurerm_resource_group.rg.name
    location                = data.azurerm_resource_group.rg.location
    set_obs                 = var.set_obs_integration
    obs_name                = var.obs_name != "" ? var.obs_name : "${var.prefix}${var.cluster_name}obs"
    obs_container_name      = var.obs_container_name != "" ? var.obs_container_name : "${var.prefix}-${var.cluster_name}-obs"
    obs_access_key          = var.blob_obs_access_key
    num_drive_containers    = var.container_number_map[var.instance_type].drive
    num_compute_containers  = var.container_number_map[var.instance_type].compute
    num_frontend_containers = var.container_number_map[var.instance_type].frontend
    nvmes_num               = var.container_number_map[var.instance_type].nvme
    tiering_ssd_percent     = var.tiering_ssd_percent
    prefix                  = var.prefix
    key_vault_uri           = azurerm_key_vault.key_vault.vault_uri
    instance_type           = var.instance_type
    install_dpdk            = var.install_cluster_dpdk
    nics_num                = var.container_number_map[var.instance_type].nics
    install_url             = var.install_weka_url != "" ? var.install_weka_url : "https://$TOKEN@get.weka.io/dist/v1/install/${var.weka_version}/${var.weka_version}"
    log_level               = var.function_app_log_level
    subnet                  = data.azurerm_subnet.subnet.address_prefix
  }

  depends_on = [azuread_service_principal.sp]
}

data "template_cloudinit_config" "management_init" {
  gzip = false
  part {
    content_type = "text/x-shellscript"
    content      = data.template_file.management-init.rendered
  }
}

resource "azurerm_public_ip" "management_public_ip" {
  name                    = "management-public-ip"
  location                = data.azurerm_resource_group.rg.location
  resource_group_name     = var.rg_name
  allocation_method       = "Dynamic"
  idle_timeout_in_minutes = 30
}

resource "azurerm_network_interface" "management_vm_nic" {
  name                = "${var.prefix}-${var.cluster_name}-management-nic"
  location            = data.azurerm_resource_group.rg.location
  resource_group_name = var.rg_name

  ip_configuration {
    primary                       = true
    name                          = "ipconfig0"
    subnet_id                     = data.azurerm_subnet.subnet.id
    private_ip_address_allocation = "Dynamic"
    public_ip_address_id          = azurerm_public_ip.management_public_ip.id
  }
}

resource "azurerm_network_interface_security_group_association" "management_nic_sg" {
  network_interface_id      = azurerm_network_interface.management_vm_nic.id
  network_security_group_id = var.sg_id
}

resource "azurerm_linux_virtual_machine" "management_vm" {
  name                = "${var.prefix}-${var.cluster_name}-management"
  computer_name       = "${var.prefix}-${var.cluster_name}-management"
  location            = data.azurerm_resource_group.rg.location
  resource_group_name = var.rg_name
  size                = "Standard_D2s_v3"
  admin_username      = var.vm_username
  custom_data         = base64encode(data.template_file.management-init.rendered)

  network_interface_ids = [
    azurerm_network_interface.management_vm_nic.id,
  ]

  os_disk {
    caching              = "ReadWrite"
    storage_account_type = "Standard_LRS"
  }

  identity {
    type = "SystemAssigned"
  }

  source_image_reference {
    publisher = "Canonical"
    offer     = "0001-com-ubuntu-server-focal"
    sku       = "20_04-lts-gen2"
    version   = "latest"
  }

  admin_ssh_key {
    username   = var.vm_username
    public_key = local.public_ssh_key
  }
}

resource "azurerm_role_assignment" "storage-blob-data-owner" {
  scope                = azurerm_storage_account.deployment_sa.id
  role_definition_name = "Storage Blob Data Owner"
  principal_id         = azuread_service_principal.sp.id
  depends_on           = [azurerm_linux_virtual_machine.management_vm, azurerm_storage_account.deployment_sa]
}

resource "azurerm_role_assignment" "storage-account-contributor" {
  scope                = data.azurerm_resource_group.rg.id
  role_definition_name = "Storage Account Contributor"
  principal_id         = azuread_service_principal.sp.id
  depends_on           = [azurerm_linux_virtual_machine.management_vm]
}

resource "azurerm_role_assignment" "mngmnt-vm-key-vault-secrets-user" {
  scope                = data.azurerm_resource_group.rg.id
  role_definition_name = "Key Vault Secrets User"
  principal_id         = azuread_service_principal.sp.id
  depends_on           = [azurerm_linux_virtual_machine.management_vm]
}

resource "azurerm_role_assignment" "mngmnt-vm-scale-set-machine-owner" {
  scope                = azurerm_linux_virtual_machine_scale_set.vmss.id
  role_definition_name = "Owner"
  principal_id         = azuread_service_principal.sp.id
  depends_on           = [azurerm_linux_virtual_machine.management_vm, azurerm_linux_virtual_machine_scale_set.vmss]
}

resource "azurerm_role_assignment" "mngmnt-vm-key-user-access-admin" {
  scope                = data.azurerm_resource_group.rg.id
  role_definition_name = "User Access Administrator"
  principal_id         = azuread_service_principal.sp.id
  depends_on           = [azurerm_linux_virtual_machine.management_vm]
}

resource "azurerm_role_assignment" "mngmnt-vm-reader" {
  scope                = data.azurerm_resource_group.rg.id
  role_definition_name = "Reader"
  principal_id         = azuread_service_principal.sp.id
  depends_on           = [azurerm_linux_virtual_machine.management_vm]
}
