locals {
  function_code_path     = "${path.module}/function-app/code"
  function_app_code_hash = md5(join("", [for f in fileset(local.function_code_path, "**") : filemd5("${local.function_code_path}/${f}")]))
  function_bins_dir      = "${path.module}/.tf-function-app"
  storage_account        = azurerm_storage_account.management_sa.name
  container_name         = azurerm_storage_container.binaries_container.name
  os                     = "darwin"
}

resource "null_resource" "upload_function_app" {
  triggers = {
    function_hash = local.function_app_code_hash
  }
  provisioner "local-exec" {
    command = <<EOT
      ${path.module}/zip_function_app_creation/create_function_binarie.sh ${local.os} ${local.function_code_path} ${local.function_bins_dir}
      ${path.module}/zip_function_app_creation/upload_to_single_sa.sh ${local.os} ${local.function_code_path} ${local.function_bins_dir} ${var.rg_name} ${local.storage_account} ${local.container_name}
      ${path.module}/zip_function_app_creation/write_function_hash_to_variables.sh ${local.os} ${local.function_code_path}
    EOT
  }
  depends_on = [azurerm_storage_account.management_sa]
}

locals {
  private_link_url = "${azurerm_storage_account.management_sa.name}.${azurerm_private_dns_zone.private_dns_zone.name}"
  token            = data.azurerm_storage_account_blob_container_sas.sa_sas.sas
  code_url         = "${local.private_link_url}/${local.container_name}/${local.function_app_code_hash}/weka-deployment${local.token}"
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
    subnets                 = join(",", data.azurerm_subnet.subnets.*.address_prefix)
  }

  depends_on = [azuread_service_principal.sp, azurerm_private_dns_zone_virtual_network_link.private_link]
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
    subnet_id                     = data.azurerm_subnet.subnets[0].id
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

  depends_on = [null_resource.upload_function_app]
}

data "http" "myip" {
  url = "https://ifconfig.me/ip"

  lifecycle {
    postcondition {
      condition     = self.status_code == 200
      error_message = "Status code invalid"
    }
  }
}

resource "azurerm_private_dns_zone" "private_dns_zone" {
  name                = "privatelink.blob.core.windows.net"
  resource_group_name = var.rg_name
}

resource "azurerm_private_endpoint" "endpoint" {
  name                = "management-endpoint"
  location            = data.azurerm_resource_group.rg.location
  resource_group_name = var.rg_name
  subnet_id           = data.azurerm_subnet.subnets[0].id

  private_service_connection {
    name                           = "management-private-connection"
    private_connection_resource_id = azurerm_storage_account.management_sa.id
    subresource_names              = ["blob"]
    is_manual_connection           = false
  }

  private_dns_zone_group {
    name                 = "private-dns-zone-group"
    private_dns_zone_ids = [azurerm_private_dns_zone.private_dns_zone.id]
  }
}

resource "azurerm_private_dns_zone_virtual_network_link" "private_link" {
  name                  = "management-private-link"
  resource_group_name   = var.rg_name
  private_dns_zone_name = azurerm_private_dns_zone.private_dns_zone.name
  virtual_network_id    = data.azurerm_virtual_network.vnet.id
}

resource "azurerm_storage_account" "management_sa" {
  name                          = "${local.alphanumeric_prefix_name}${local.alphanumeric_cluster_name}management"
  location                      = data.azurerm_resource_group.rg.location
  resource_group_name           = var.rg_name
  account_kind                  = "StorageV2"
  account_tier                  = "Standard"
  account_replication_type      = "ZRS"
  public_network_access_enabled = true
  enable_https_traffic_only     = false
}

resource "azurerm_storage_account_network_rules" "management_sa_net_rules" {
  storage_account_id = azurerm_storage_account.management_sa.id

  default_action             = "Deny"
  ip_rules                   = [chomp(data.http.myip.response_body)]
  virtual_network_subnet_ids = [data.azurerm_subnet.subnets[0].id]
  bypass                     = []

  private_link_access {
    endpoint_resource_id = azurerm_private_endpoint.endpoint.id
  }
}

resource "azurerm_storage_container" "binaries_container" {
  name                  = "${local.alphanumeric_prefix_name}${local.alphanumeric_cluster_name}-binaries"
  storage_account_name  = azurerm_storage_account.management_sa.name
  container_access_type = "private"
  depends_on            = [azurerm_storage_account.management_sa]
}


data "azurerm_storage_account_blob_container_sas" "sa_sas" {
  connection_string = azurerm_storage_account.management_sa.primary_connection_string
  container_name    = azurerm_storage_container.binaries_container.name
  https_only        = false

  start  = timestamp()
  expiry = timeadd("${timestamp()}", "40m")

  permissions {
    read   = true
    add    = true
    create = false
    write  = false
    delete = false
    list   = true
  }
}

# resource "azurerm_role_assignment" "subscription-reader" {
#   scope                = "${data.azurerm_subscription.primary.id}"
#   role_definition_name = "Reader"
#   principal_id         = "${azuread_service_principal.sp.id}"
# }

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
  scope                = var.custom_image_id != null ? azurerm_linux_virtual_machine_scale_set.custom_image_vmss[0].id : azurerm_linux_virtual_machine_scale_set.default_image_vmss[0].id
  role_definition_name = "Owner"
  principal_id         = azuread_service_principal.sp.id
  depends_on           = [azurerm_linux_virtual_machine.management_vm, azurerm_linux_virtual_machine_scale_set.custom_image_vmss, azurerm_linux_virtual_machine_scale_set.default_image_vmss]
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
