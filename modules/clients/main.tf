data "azurerm_resource_group" "rg" {
  name = var.rg_name
}

data "azurerm_subnet" "subnet" {
  resource_group_name  = var.vnet_rg_name
  virtual_network_name = var.vnet_name
  name                 = var.subnet_name
}

locals {
  first_nic_ids = var.assign_public_ip ? azurerm_network_interface.public_first_nic[*].id : azurerm_network_interface.private_first_nic[*].id
  nics_num      = var.frontend_container_cores_num + 1
  preparation_script = templatefile("${path.module}/init.sh", {
    apt_repo_server = var.apt_repo_server
    nics_num        = local.nics_num
    subnet_range    = data.azurerm_subnet.subnet.address_prefix
  })

  mount_wekafs_script = templatefile("${path.module}/mount_wekafs.sh", {
    all_gateways                 = cidrhost(data.azurerm_subnet.subnet.address_prefix, 1)
    frontend_container_cores_num = var.frontend_container_cores_num
    backend_lb_ip                = var.backend_lb_ip
    clients_use_dpdk             = var.clients_use_dpdk
    rg_name                      = var.rg_name
    vmss_name                    = var.vmss_name
  })

  custom_data_parts = [local.preparation_script, local.mount_wekafs_script, "${var.custom_data}\n"]
  vms_custom_data   = base64encode(join("\n", local.custom_data_parts))

  client_identity_id = var.vm_identity_name == "" ? azurerm_user_assigned_identity.this[0].id : data.azurerm_user_assigned_identity.this[0].id

  arm_instances = ["Standard_D4ps_v5", "Standard_D8ps_v5", "Standard_D16ps_v5", "Standard_D32ps_v5", "Standard_D48ps_v5", "Standard_D64ps_v5", "Standard_D8plds_v5", "Standard_D32plds_v5", "Standard_D64plds_v5"]
  default_arch  = contains(local.arm_instances, var.instance_type) ? "arm64" : "x86_64"
  arch          = var.arch == null ? local.default_arch : var.arch
  default_client_instance_type = {
    x86_64 = "Standard_D8_v5"
    arm64  = "Standard_E2ps_v5"
  }
  instance_type = var.instance_type != "" ? var.instance_type : local.default_client_instance_type[local.arch]
  default_source_image_id = {
    x86_64 = "/communityGalleries/WekaIO-d7d3f308-d5a1-4c45-8e8a-818aed57375a/images/ubuntu20.04/versions/latest"
    arm64  = "/communityGalleries/WekaIO-d7d3f308-d5a1-4c45-8e8a-818aed57375a/images/arm20.04/versions/latest"
  }
  source_image_id = var.source_image_id != "" ? var.source_image_id : local.default_source_image_id[local.arch]
}

resource "azurerm_public_ip" "public_ip" {
  count               = var.assign_public_ip ? var.clients_number : 0
  name                = "${var.clients_name}-public-ip-${count.index}"
  resource_group_name = var.rg_name
  location            = data.azurerm_resource_group.rg.location
  allocation_method   = "Static"
  sku                 = "Standard"
  tags                = var.tags_map
}

resource "azurerm_network_interface" "public_first_nic" {
  count                         = var.assign_public_ip ? var.clients_number : 0
  name                          = "${var.clients_name}-backend-nic-${count.index}"
  enable_accelerated_networking = var.clients_use_dpdk
  resource_group_name           = var.rg_name
  location                      = data.azurerm_resource_group.rg.location
  tags                          = var.tags_map
  ip_configuration {
    name                          = "ipconfig0"
    subnet_id                     = data.azurerm_subnet.subnet.id
    private_ip_address_allocation = "Dynamic"
    primary                       = true
    public_ip_address_id          = azurerm_public_ip.public_ip[count.index].id
  }
}

resource "azurerm_network_interface_security_group_association" "public_first" {
  count                     = var.assign_public_ip ? var.clients_number : 0
  network_interface_id      = azurerm_network_interface.public_first_nic[count.index].id
  network_security_group_id = var.sg_id
}

resource "azurerm_network_interface" "private_first_nic" {
  count                         = var.assign_public_ip ? 0 : var.clients_number
  name                          = "${var.clients_name}-backend-nic-${count.index}"
  enable_accelerated_networking = var.clients_use_dpdk
  resource_group_name           = var.rg_name
  location                      = data.azurerm_resource_group.rg.location
  tags                          = var.tags_map
  ip_configuration {
    name                          = "ipconfig0"
    subnet_id                     = data.azurerm_subnet.subnet.id
    private_ip_address_allocation = "Dynamic"
    primary                       = true
  }
}

resource "azurerm_network_interface_security_group_association" "private_first" {
  count                     = var.assign_public_ip ? 0 : var.clients_number
  network_interface_id      = azurerm_network_interface.private_first_nic[count.index].id
  network_security_group_id = var.sg_id
}

resource "azurerm_network_interface" "private_nics" {
  count                         = (local.nics_num - 1) * var.clients_number
  name                          = "${var.clients_name}-backend-nic-${count.index + var.clients_number}"
  enable_accelerated_networking = var.clients_use_dpdk
  resource_group_name           = var.rg_name
  location                      = data.azurerm_resource_group.rg.location
  tags                          = var.tags_map
  ip_configuration {
    name                          = "ipconfig${count.index + var.clients_number}"
    subnet_id                     = data.azurerm_subnet.subnet.id
    private_ip_address_allocation = "Dynamic"
  }
}

resource "azurerm_network_interface_security_group_association" "private" {
  count                     = (local.nics_num - 1) * var.clients_number
  network_interface_id      = azurerm_network_interface.private_nics[count.index].id
  network_security_group_id = var.sg_id
}

resource "azurerm_linux_virtual_machine" "this" {
  count               = var.clients_number
  name                = "${var.clients_name}-vm-${count.index}"
  location            = data.azurerm_resource_group.rg.location
  resource_group_name = var.rg_name
  admin_username      = var.vm_username
  tags                = merge({ "weka_cluster_client" : var.clients_name }, var.tags_map)
  custom_data         = local.vms_custom_data
  source_image_id     = local.source_image_id
  size                = local.instance_type
  network_interface_ids = concat([
    local.first_nic_ids[count.index]
  ], slice(azurerm_network_interface.private_nics[*].id, (local.nics_num - 1) * count.index, (local.nics_num - 1) * (count.index + 1)))

  proximity_placement_group_id    = var.ppg_id
  disable_password_authentication = true

  identity {
    type         = "UserAssigned"
    identity_ids = [local.client_identity_id]
  }

  os_disk {
    caching              = "ReadWrite"
    name                 = "${var.clients_name}-os-disk-${count.index}"
    storage_account_type = "StandardSSD_LRS"
  }

  admin_ssh_key {
    public_key = var.ssh_public_key
    username   = var.vm_username
  }
  lifecycle {
    ignore_changes = [tags, custom_data]
  }
  depends_on = [azurerm_network_interface.private_first_nic, azurerm_network_interface.private_nics, azurerm_network_interface.public_first_nic]
}
