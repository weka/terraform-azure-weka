resource "azurerm_resource_group" "rg" {
  location = var.location
  name     = "${var.prefix}-apt-repo-rg"
}

resource "azurerm_virtual_network" "vnet" {
  name                = "${var.prefix}-apt-repo-vnet"
  address_space       = var.address_space
  location            = var.location
  resource_group_name = azurerm_resource_group.rg.name
}

resource "azurerm_subnet" "subnets" {
  count                = length(var.address_prefixes)
  name                 = "${var.prefix}-apt-repo-subnet-${count.index}"
  resource_group_name  = azurerm_resource_group.rg.name
  virtual_network_name = azurerm_virtual_network.vnet.name
  address_prefixes     = [var.address_prefixes[count.index]]
}

resource "azurerm_public_ip" "public_ip" {
  name                = "${var.prefix}-repo-public-ip"
  location            = azurerm_resource_group.rg.location
  resource_group_name = azurerm_resource_group.rg.name
  allocation_method   = "Static"
  sku                 = "Standard"
  domain_name_label   = "${var.prefix}-apt-repo"
  lifecycle {
    ignore_changes = [tags, zones, ip_tags]
  }
}

resource "azurerm_network_security_group" "nsg" {
  name                = "${var.prefix}-apt-repo-nsg"
  location            = azurerm_resource_group.rg.location
  resource_group_name = azurerm_resource_group.rg.name

  security_rule {
    name                       = "SSH"
    priority                   = 1001
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "22"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }
  security_rule {
    access                     = "Allow"
    direction                  = "Inbound"
    name                       = "http"
    priority                   = 1002
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "80"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }
}

resource "azurerm_network_interface_security_group_association" "sg_association" {
  network_interface_id      = azurerm_network_interface.vm_interface.id
  network_security_group_id = azurerm_network_security_group.nsg.id
}

resource "azurerm_virtual_machine" "apt_repo_vm_linux" {
  name                          = "${var.prefix}-apt-repo-vm"
  location                      = azurerm_resource_group.rg.location
  resource_group_name           = azurerm_resource_group.rg.name
  vm_size                       = var.vm_size
  network_interface_ids         = [azurerm_network_interface.vm_interface.id]
  delete_os_disk_on_termination = true

  storage_image_reference {
    publisher = "Canonical"
    offer     = "0001-com-ubuntu-server-jammy"
    sku       = "22_04-lts-gen2"
    version   = "latest"
  }

  storage_os_disk {
    name              = "${var.prefix}-apt-repo-osdisk"
    create_option     = "FromImage"
    caching           = "ReadWrite"
    managed_disk_type = "Standard_LRS"
    disk_size_gb      = 500
  }

  os_profile {
    computer_name  = "${var.prefix}-apt-repo-vm"
    admin_username = var.admin_username
    custom_data    = file("init.sh")
  }

  os_profile_linux_config {
    disable_password_authentication = true

    ssh_keys {
      path     = "/home/${var.admin_username}/.ssh/authorized_keys"
      key_data = file(var.ssh_public_key)
    }
  }
  depends_on = [azurerm_network_interface.vm_interface, azurerm_network_security_group.nsg]
}

resource "azurerm_network_interface" "vm_interface" {
  name                          = "${var.prefix}-vm-nic"
  location                      = azurerm_resource_group.rg.location
  resource_group_name           = azurerm_resource_group.rg.name
  enable_accelerated_networking = false

  ip_configuration {
    name                          = "ipconfig"
    subnet_id                     = azurerm_subnet.subnets[0].id
    private_ip_address_allocation = "Dynamic"
    public_ip_address_id          = azurerm_public_ip.public_ip.id
  }
  depends_on = [azurerm_public_ip.public_ip]
}
