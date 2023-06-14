output "vnet-name" {
  value = var.vnet_name != null ? var.vnet_name : azurerm_virtual_network.vnet[0].name
  description = "Displays the VNet name."
}

output "subnets-name" {
  value = var.subnet_name == null ? azurerm_subnet.subnet[0].name : var.subnet_name
  description = "Displays the subnet names list."
}

output "sg-id" {
  value = azurerm_network_security_group.sg.id
  description = "Displays the security group id."
}

output "private-dns-zone-name" {
  value = var.create_private_dns_zone ? azurerm_private_dns_zone.dns[0].name : null
  description = "Displays the private DNS zone name."
}

output "vnet_rg_name" {
  value = var.vnet_rg_name == null ? var.rg_name : var.vnet_rg_name
  description = "Resource group name of vnet."
}
