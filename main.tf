data azurerm_resource_group "rg"{
  name = var.rg_name
}

locals {
  private_nic_first_index = var.private_network ? 0 : 1
}

data "azurerm_subnet" "subnets" {
  count                = length(var.subnets)
  resource_group_name  = var.rg_name
  virtual_network_name = var.vnet_name
  name                 = var.subnets[count.index]
}

data "azurerm_virtual_network" "vnet" {
  name = var.vnet_name
  resource_group_name = var.rg_name
}

# ===================== SSH key ++++++++++++++++++++++++= #
locals {
  ssh_path = "/tmp/${var.prefix}-${var.cluster_name}"
}

resource "tls_private_key" "ssh_key" {
  count     = var.ssh_public_key == null ? 1 : 0
  algorithm = "RSA"
  rsa_bits  = 4096
}

resource "local_file" "public_key" {
  count           = var.ssh_public_key == null ? 1 : 0
  content         = tls_private_key.ssh_key[count.index].public_key_openssh
  filename        = "${local.ssh_path}-public-key.pub"
  file_permission = "0600"
}

resource "local_file" "private_key" {
  count           = var.ssh_public_key == null ? 1 : 0
  content         = tls_private_key.ssh_key[count.index].private_key_pem
  filename        = "${local.ssh_path}-private-key.pem"
  file_permission = "0600"
}

locals {
  public_ssh_key  = var.ssh_public_key == null ? tls_private_key.ssh_key[0].public_key_openssh : file(var.ssh_public_key)
  private_ssh_key = var.ssh_private_key == null ? tls_private_key.ssh_key[0].private_key_pem : file(var.ssh_private_key)
  alphanumeric_cluster_name =  lower(replace(var.cluster_name,"/\\W|_|\\s/",""))
  alphanumeric_prefix_name  = lower(replace(var.prefix,"/\\W|_|\\s/",""))
}

# ==================== Backend VMs ======================= #
resource "azurerm_public_ip" "vm-ip" {
  count               = var.private_network ? 0 : var.cluster_size
  name                = "${var.prefix}-${var.cluster_name}-backend-ip-${count.index}"
  location            = data.azurerm_resource_group.rg.location
  resource_group_name = var.rg_name
  allocation_method   = "Static"
  sku                 = "Standard"
  domain_name_label   = "${local.alphanumeric_prefix_name}-${local.alphanumeric_cluster_name}-backend-${count.index}"
  lifecycle {
    ignore_changes = [tags,zones,ip_tags]
  }
  tags               = merge(var.tags_map, {"weka_cluster": var.cluster_name})
}

resource "azurerm_network_interface" "primary-nic" {
  count                         = var.cluster_size
  name                          = "${local.alphanumeric_prefix_name}-${local.alphanumeric_cluster_name}-backend-${count.index}-nic"
  location                      = data.azurerm_resource_group.rg.location
  resource_group_name           = var.rg_name
  enable_accelerated_networking = false
  internal_dns_name_label       = "${local.alphanumeric_prefix_name}-${local.alphanumeric_cluster_name}-backend-${count.index}"
  tags                          = merge(var.tags_map, {"weka_cluster": var.cluster_name})
  dynamic "ip_configuration" {
    for_each = range(local.private_nic_first_index)
    content {
      primary                       = true
      name                          = "ipconfig1"
      private_ip_address_allocation = "Dynamic"
      subnet_id                     = data.azurerm_subnet.subnets[0].id
      public_ip_address_id          = azurerm_public_ip.vm-ip[count.index].id
    }
  }
  dynamic "ip_configuration" {
    for_each = range(local.private_nic_first_index,1)
    content {
      primary                       = true
      name                          = "ipconfig1"
      private_ip_address_allocation = "Dynamic"
      subnet_id                     = data.azurerm_subnet.subnets[0].id
    }
  }
  lifecycle {
    ignore_changes = [ip_configuration, tags]
  }
  depends_on = [azurerm_public_ip.vm-ip]
}

resource "azurerm_availability_set" "availability_set" {
  name                         = "${var.prefix}-${var.cluster_name}-backend-availability-set"
  location                     = data.azurerm_resource_group.rg.location
  resource_group_name          = var.rg_name
  proximity_placement_group_id = azurerm_proximity_placement_group.ppg.id
  tags                         = merge(var.tags_map, {"weka_cluster": var.cluster_name})
  depends_on                   = [azurerm_proximity_placement_group.ppg]
}

resource "azurerm_network_interface_security_group_association" "sg-association" {
  count                     = var.cluster_size
  network_interface_id      = azurerm_network_interface.primary-nic[count.index].id
  network_security_group_id = var.sg_id
  depends_on                = [azurerm_network_interface.primary-nic]
}

data "template_file" "init" {
  template = file("${path.module}/user-data.sh")

  vars = {
    weka_token           = var.get_weka_io_token
    weka_version         = var.weka_version
    install_weka_url     = var.install_weka_url
    apt_repo_url         = var.apt_repo_url
    private_ssh_key      = local.private_ssh_key
    user                 = var.vm_username
    ofed_version         = var.ofed_version
    install_ofed_url     = var.install_ofed_url
    num_drive_containers = var.container_number_map[var.instance_type].drive
    clusterization_url = "https://${var.prefix}-${var.cluster_name}-function-app.azurewebsites.net/api/clusterize"
    function_app_default_key = data.azurerm_function_app_host_keys.function_keys.default_function_key
  }
}

data "template_cloudinit_config" "cloud_init" {
  gzip          = false
  part {
    content_type = "text/x-shellscript"
    content      = data.template_file.init.rendered
  }
}

resource "azurerm_proximity_placement_group" "ppg" {
  name                = "${var.prefix}-${var.cluster_name}-backend-ppg"
  location            = data.azurerm_resource_group.rg.location
  resource_group_name = var.rg_name
  tags                = merge(var.tags_map, {"weka_cluster": var.cluster_name})
}

resource "azurerm_linux_virtual_machine" "vm" {
  count                            = var.cluster_size
  name                             = local.vm_names[count.index]
  location                         = data.azurerm_resource_group.rg.location
  resource_group_name              = var.rg_name
  network_interface_ids            = [azurerm_network_interface.primary-nic[count.index].id]
  size                             = var.instance_type
  availability_set_id              = azurerm_availability_set.availability_set.id
  proximity_placement_group_id     = azurerm_proximity_placement_group.ppg.id
  source_image_reference {
    offer     = lookup(var.linux_vm_image, "offer", null)
    publisher = lookup(var.linux_vm_image, "publisher", null)
    sku       = lookup(var.linux_vm_image, "sku", null)
    version   = lookup(var.linux_vm_image, "version", null)
  }
  os_disk {
    name                 = "${var.prefix}-${var.cluster_name}-backend-os-disk-${count.index}"
    caching              = "ReadWrite"
    storage_account_type = "StandardSSD_LRS"
  }
  identity {
    type = "SystemAssigned"
  }
  custom_data                     =  base64encode(data.template_file.init.rendered)
  computer_name                   = "${local.alphanumeric_prefix_name}-${local.alphanumeric_cluster_name}-backend-${count.index}"
  admin_username                  = var.vm_username
  disable_password_authentication = true
  tags                            = merge(var.tags_map, {"weka_cluster": var.cluster_name})
  admin_ssh_key {
    username   = var.vm_username
    public_key = local.public_ssh_key
  }
  lifecycle {
    ignore_changes = [custom_data]
  }
  depends_on = [azurerm_network_interface.primary-nic, azurerm_public_ip.vm-ip]
}

resource "azurerm_managed_disk" "managed_disk" {
  count                = length(azurerm_linux_virtual_machine.vm.*.id)
  name                 = "${var.prefix}-${var.cluster_name}-disk-${count.index}"
  location             = data.azurerm_resource_group.rg.location
  resource_group_name  = var.rg_name
  storage_account_type = "StandardSSD_LRS"
  create_option        = "Empty"
  disk_size_gb         = local.disk_size
}

resource "azurerm_virtual_machine_data_disk_attachment" "data_disk_attachment" {
  count              = length(azurerm_linux_virtual_machine.vm.*.id)
  managed_disk_id    = azurerm_managed_disk.managed_disk[count.index].id
  virtual_machine_id = azurerm_linux_virtual_machine.vm[count.index].id
  lun                = "10"
  caching            = "ReadWrite"
}


locals {
  vm_names = [for i in range(var.cluster_size): "${var.prefix}-${var.cluster_name}-backend-${i}"]
  disk_size = var.default_disk_size + var.traces_per_ionode * (var.container_number_map[var.instance_type].compute + var.container_number_map[var.instance_type].drive + var.container_number_map[var.instance_type].frontend)
}

