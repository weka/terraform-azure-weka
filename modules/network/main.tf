data "azurerm_resource_group" "rg" {
  name = var.rg_name
}

data "azurerm_resource_group" "vnet_rg" {
  count = var.vnet_rg_name != "" ? 1 : 0
  name  = var.vnet_rg_name
}

locals {
  vnet_rg             = var.vnet_rg_name == "" ? data.azurerm_resource_group.rg.name : var.vnet_rg_name
  private_dns_rg_name = var.private_dns_rg_name == "" ? data.azurerm_resource_group.rg.name : var.private_dns_rg_name
  vnet_rg_location    = var.vnet_rg_name == "" ? data.azurerm_resource_group.rg.location : data.azurerm_resource_group.vnet_rg[0].location
  vnet_name           = var.vnet_name == "" ? azurerm_virtual_network.vnet[0].name : var.vnet_name
}

resource "azurerm_virtual_network" "vnet" {
  count               = var.vnet_name == "" ? 1 : 0
  resource_group_name = local.vnet_rg
  location            = local.vnet_rg_location
  name                = "${var.prefix}-vnet"
  address_space       = [var.address_space]
  tags                = merge(var.tags_map)
  depends_on          = [data.azurerm_resource_group.rg]
  lifecycle {
    ignore_changes = [tags]
  }
}

data "azurerm_virtual_network" "vnet_data" {
  count               = var.vnet_name != "" ? 1 : 0
  name                = var.vnet_name
  resource_group_name = local.vnet_rg
}

resource "azurerm_subnet" "subnet" {
  count                = var.subnet_name == "" ? 1 : 0
  resource_group_name  = local.vnet_rg
  name                 = "${var.prefix}-subnet-${count.index}"
  address_prefixes     = [var.subnet_prefix]
  virtual_network_name = local.vnet_name
  lifecycle {
    ignore_changes = [service_endpoint_policy_ids, service_endpoints]
  }
  depends_on = [data.azurerm_resource_group.rg, azurerm_virtual_network.vnet]
}

# ====================== sg ssh ========================== #
resource "azurerm_network_security_rule" "sg_public_ssh" {
  count                       = length(var.allow_ssh_ranges)
  name                        = "${var.prefix}-ssh-sg-${count.index}"
  resource_group_name         = data.azurerm_resource_group.rg.name
  priority                    = 100 + (count.index + 1)
  direction                   = "Inbound"
  access                      = "Allow"
  protocol                    = "Tcp"
  source_port_range           = "*"
  destination_port_range      = "22"
  source_address_prefix       = element(var.allow_ssh_ranges, count.index)
  destination_address_prefix  = "*"
  network_security_group_name = azurerm_network_security_group.sg.name
}

# ====================== sg  ========================== #
resource "azurerm_network_security_rule" "sg_weka_ui" {
  count                       = length(var.allow_weka_api_ranges)
  name                        = "${var.prefix}-ui-sg-${count.index}"
  resource_group_name         = data.azurerm_resource_group.rg.name
  priority                    = 200 + (count.index + 1)
  direction                   = "Inbound"
  access                      = "Allow"
  protocol                    = "Tcp"
  source_port_range           = "*"
  destination_port_range      = "14000"
  source_address_prefix       = element(var.allow_weka_api_ranges, count.index)
  destination_address_prefix  = "*"
  network_security_group_name = azurerm_network_security_group.sg.name
}

resource "azurerm_network_security_group" "sg" {
  name                = "${var.prefix}-sg"
  location            = data.azurerm_resource_group.rg.location
  resource_group_name = data.azurerm_resource_group.rg.name
  tags                = merge(var.tags_map)

  lifecycle {
    ignore_changes = [tags]
  }
  depends_on = [data.azurerm_resource_group.rg]
}

resource "azurerm_subnet_network_security_group_association" "sg_association" {
  count                     = var.subnet_name == "" ? 1 : 0
  subnet_id                 = azurerm_subnet.subnet[count.index].id
  network_security_group_id = azurerm_network_security_group.sg.id
  depends_on                = [azurerm_network_security_group.sg]
}

# ================== Private DNS ========================= #
resource "azurerm_private_dns_zone" "dns" {
  count               = var.private_dns_zone_name == "" ? 1 : 0
  name                = "${var.prefix}.private.net"
  resource_group_name = local.private_dns_rg_name
  tags                = merge(var.tags_map)
  lifecycle {
    ignore_changes = [tags]
  }
}

resource "azurerm_private_dns_zone_virtual_network_link" "dns_vnet_link" {
  count                 = var.private_dns_zone_name == "" ? 1 : 0
  name                  = "${var.prefix}-private-network-link"
  resource_group_name   = data.azurerm_resource_group.rg.name
  private_dns_zone_name = azurerm_private_dns_zone.dns[0].name
  virtual_network_id    = var.vnet_name != "" ? data.azurerm_virtual_network.vnet_data[0].id : azurerm_virtual_network.vnet[0].id
  registration_enabled  = true
  lifecycle {
    ignore_changes = [tags]
  }
}
