data "azurerm_resource_group" "rg" {
  name = var.rg_name
}

data "azurerm_resource_group" "vnet_rg" {
  count = var.vnet_rg_name != null ? 1 : 0
  name  = var.vnet_rg_name
}

locals {
  vnet_rg          = var.vnet_rg_name == null ? data.azurerm_resource_group.rg.name : var.vnet_rg_name
  vnet_rg_location = var.vnet_rg_name == null ? data.azurerm_resource_group.rg.location : data.azurerm_resource_group.vnet_rg[0].location
  vnet_name        = var.vnet_name == null ? azurerm_virtual_network.vnet[0].name : data.azurerm_virtual_network.vnet_data[0].name
}

resource "azurerm_virtual_network" "vnet" {
  count               = var.vnet_name == null ? 1 : 0
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
  count               = var.vnet_name != null  ? 1 : 0
  name                = var.vnet_name
  resource_group_name = local.vnet_rg
}

resource "azurerm_subnet" "subnet" {
  count                = length(var.subnets_name_list) == 0 ? length(var.subnet_prefixes) : 0
  resource_group_name  = local.vnet_rg
  name                 = "${var.prefix}-subnet-${count.index}"
  address_prefixes     = [var.subnet_prefixes[count.index]]
  virtual_network_name = local.vnet_name
  service_endpoints    = ["Microsoft.Web", "Microsoft.KeyVault","Microsoft.Storage"]

  lifecycle {
    ignore_changes = [service_endpoint_policy_ids, service_endpoints]
  }
  depends_on = [data.azurerm_resource_group.rg, azurerm_virtual_network.vnet]
}

resource "azurerm_subnet" "subnet-delegation" {
  count                = length(var.subnets_delegation_names) == 0 ? length(var.subnets_delegation_prefixes) : 0
  name                 = "${var.prefix}-subnet-delegation-${count.index}"
  resource_group_name  = var.rg_name
  virtual_network_name = local.vnet_name
  address_prefixes     = [var.subnets_delegation_prefixes[count.index]]
  service_endpoints    = ["Microsoft.Web", "Microsoft.KeyVault","Microsoft.Storage"]

  delegation {
    name = "subnet-delegation-${count.index}"
    service_delegation {
      name    = "Microsoft.Web/serverFarms"
      actions = ["Microsoft.Network/virtualNetworks/subnets/action"]
    }
  }
}

data "azurerm_subnet" "subnets_data" {
  count                = length(var.subnets_name_list) > 0 ? length(var.subnets_name_list) : 0
  name                 = var.subnets_name_list[count.index]
  resource_group_name  = local.vnet_rg
  virtual_network_name = var.vnet_name != null ? var.vnet_name : azurerm_virtual_network.vnet[0].name
}

# ====================== sg ssh ========================== #
resource "azurerm_network_security_rule" "sg_public_ssh" {
  count                       = var.private_network ? 0 : length(var.sg_public_ssh_ips)
  name                        = "${var.prefix}-ssh-sg-${count.index}"
  resource_group_name         = data.azurerm_resource_group.rg.name
  priority                    = "100${count.index}"
  direction                   = "Inbound"
  access                      = "Allow"
  protocol                    = "Tcp"
  source_port_range           = "*"
  destination_port_range      = "22"
  source_address_prefixes     = [var.sg_public_ssh_ips[count.index]]
  destination_address_prefix  = "*"
  network_security_group_name = azurerm_network_security_group.sg.name
}

# ====================== sg  ========================== #
resource "azurerm_network_security_rule" "sg_weka_ui" {
  count                       = var.private_network ? 0 : 1
  name                        = "${var.prefix}-ui-sg"
  resource_group_name         = data.azurerm_resource_group.rg.name
  priority                    = "1002"
  direction                   = "Inbound"
  access                      = "Allow"
  protocol                    = "Tcp"
  source_port_range           = "*"
  destination_port_range      = "14000"
  source_address_prefix       = "*"
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

resource "azurerm_subnet_network_security_group_association" "sg-association" {
  count                     = length(var.subnets_name_list) > 0 ? length(var.subnets_name_list) : length(var.subnet_prefixes)
  subnet_id                 = length(var.subnets_name_list) > 0 ? data.azurerm_subnet.subnets_data[count.index].id : azurerm_subnet.subnet[count.index].id
  network_security_group_id = azurerm_network_security_group.sg.id
  depends_on                = [azurerm_network_security_group.sg]
}

resource "azurerm_subnet_network_security_group_association" "delegation-sg-association" {
  count                     = length(var.subnets_delegation_names) == 0 ? length(var.subnets_delegation_prefixes) : 0
  subnet_id                 = azurerm_subnet.subnet-delegation[count.index].id
  network_security_group_id = azurerm_network_security_group.sg.id
  depends_on                = [azurerm_network_security_group.sg]
}

# ================== Private DNS ========================= #
resource "azurerm_private_dns_zone" "dns" {
  count               = var.create_private_dns_zone ? 1 : 0
  name                = "${var.prefix}.private.net"
  resource_group_name = data.azurerm_resource_group.rg.name
  tags                = merge(var.tags_map)
  lifecycle {
    ignore_changes = [tags]
  }
}

resource "azurerm_private_dns_zone_virtual_network_link" "dns_vnet_link" {
  count                 = var.create_private_dns_zone ? 1 : 0
  name                  = "${var.prefix}-private-network-link"
  resource_group_name   = data.azurerm_resource_group.rg.name
  private_dns_zone_name = azurerm_private_dns_zone.dns[0].name
  virtual_network_id    = var.vnet_name != null ? data.azurerm_virtual_network.vnet_data[0].id : azurerm_virtual_network.vnet[0].id
  registration_enabled  = true
  lifecycle {
    ignore_changes = [tags]
  }
}

resource "azurerm_private_dns_zone" "sites_dns" {
  count               = var.sites_dns_zone_name == null ? 1 : 0
  name                = "privatelink.azurewebsites.net"
  resource_group_name = data.azurerm_resource_group.rg.name
  tags                = merge(var.tags_map)
  lifecycle {
    ignore_changes = [tags]
  }
}

resource "azurerm_private_dns_zone_virtual_network_link" "sites_dns_vnet_link" {
  count                 = var.sites_dns_zone_link_to_network == false ? 0 : 1
  name                  = "${var.prefix}-private-sites-link"
  resource_group_name   = var.rg_name
  private_dns_zone_name = var.sites_dns_zone_name == null ? azurerm_private_dns_zone.sites_dns[0].name : var.sites_dns_zone_name
  virtual_network_id    = var.vnet_name != null ? data.azurerm_virtual_network.vnet_data[0].id : azurerm_virtual_network.vnet[0].id
  registration_enabled  = false
}

# Create Private DNS Zone
resource "azurerm_private_dns_zone" "blob-dns-zone" {
  count               = var.private_dns_blob_name == null ? 1 : 0
  name                = "privatelink.blob.core.windows.net"
  resource_group_name = var.rg_name
  tags                = merge(var.tags_map)
  lifecycle {
    ignore_changes = [tags]
  }
}

resource "azurerm_private_dns_zone_virtual_network_link" "blob_dns_vnet_link" {
  count                 = var.blob_dns_zone_link_to_network == false ? 0 : 1
  name                  = "${var.prefix}-private-blob-link"
  resource_group_name   = var.rg_name
  private_dns_zone_name = var.sites_dns_zone_name == null ? azurerm_private_dns_zone.blob-dns-zone[0].name : var.private_dns_blob_name
  virtual_network_id    = var.vnet_name != null ? data.azurerm_virtual_network.vnet_data[0].id : azurerm_virtual_network.vnet[0].id
  registration_enabled  = false
}


# Create Private DNS Zone
resource "azurerm_private_dns_zone" "keyvault-dns-zone" {
  count               = var.keyvault_dns_zone_name == null ? 1 : 0
  name                = "privatelink.vaultcore.azure.net"
  resource_group_name = var.rg_name
  tags                = merge(var.tags_map)
  lifecycle {
    ignore_changes = [tags]
  }
}

resource "azurerm_private_dns_zone_virtual_network_link" "keyvault_dns_vnet_link" {
  count                 = var.keyvault_dns_zone_link_to_network == false ? 0 : 1
  name                  = "${var.prefix}-private-blob-link"
  resource_group_name   = var.rg_name
  private_dns_zone_name = var.keyvault_dns_zone_name == null ? azurerm_private_dns_zone.keyvault-dns-zone[0].name : var.keyvault_dns_zone_name
  virtual_network_id    = var.vnet_name != null ? data.azurerm_virtual_network.vnet_data[0].id : azurerm_virtual_network.vnet[0].id
  registration_enabled  = false
}
