
locals {
  ssh_path                  = "/tmp/${var.prefix}-${var.cluster_name}"
  ssh_public_key_path       = "${local.ssh_path}-public-key.pub"
  ssh_private_key_path      = "${local.ssh_path}-private-key.pem"
  public_ssh_key            = var.ssh_public_key == null ? tls_private_key.ssh_key[0].public_key_openssh : var.ssh_public_key
  disk_size                 = var.default_disk_size + var.traces_per_ionode * (var.containers_config_map[var.instance_type].compute + var.containers_config_map[var.instance_type].drive + var.containers_config_map[var.instance_type].frontend)
  alphanumeric_cluster_name = lower(replace(var.cluster_name, "/\\W|_|\\s/", ""))
  alphanumeric_prefix_name  = lower(replace(var.prefix, "/\\W|_|\\s/", ""))
  subnet_range              = data.azurerm_subnet.subnet.address_prefix
  nics_numbers              = var.install_cluster_dpdk ? var.containers_config_map[var.instance_type].nics : 1
  init_script = templatefile("${path.module}/init-script.sh", {
    apt_repo_server          = var.apt_repo_server
    user                     = var.vm_username
    subnet_range             = local.subnet_range
    nics_num                 = local.nics_numbers
    deploy_url               = "https://${azurerm_linux_function_app.function_app.name}.azurewebsites.net/api/deploy"
    report_url               = "https://${azurerm_linux_function_app.function_app.name}.azurewebsites.net/api/report"
    function_app_default_key = data.azurerm_function_app_host_keys.function_keys.default_function_key
    disk_size                = local.disk_size
  })
  custom_data_script = templatefile("${path.module}/user-data.sh", {
    user_data   = var.user_data
    init_script = local.init_script
  })
  placement_group_id = var.placement_group_id != "" ? var.placement_group_id : var.vmss_single_placement_group ? azurerm_proximity_placement_group.ppg[0].id : null
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
  count               = var.placement_group_id == "" && var.vmss_single_placement_group ? 1 : 0
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

resource "azurerm_user_assigned_identity" "vmss" {
  location            = data.azurerm_resource_group.rg.location
  name                = "${var.prefix}-${var.cluster_name}-vm-identity"
  resource_group_name = data.azurerm_resource_group.rg.name
}

resource "azurerm_role_assignment" "reader" {
  scope                = data.azurerm_resource_group.rg.id
  role_definition_name = "Reader"
  principal_id         = azurerm_user_assigned_identity.vmss.principal_id
}

resource "azurerm_role_assignment" "network_contributor" {
  scope                = data.azurerm_resource_group.rg.id
  role_definition_name = "Network Contributor"
  principal_id         = azurerm_user_assigned_identity.vmss.principal_id
}

resource "azurerm_role_assignment" "storage_blob_data_reader" {
  count                = var.weka_tar_storage_account_id != "" ? 1 : 0
  scope                = var.weka_tar_storage_account_id
  role_definition_name = "Storage Blob Data Reader"
  principal_id         = azurerm_user_assigned_identity.vmss.principal_id
}
