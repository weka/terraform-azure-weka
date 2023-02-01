output "vnet-name" {
  value = var.vnet_name != null ? var.vnet_name : azurerm_virtual_network.vnet[0].name
  description = "Displays the VNet name."
}

output "subnets-name" {
  value = length(var.subnets_name_list) > 0 ? var.subnets_name_list : azurerm_subnet.subnet.*.name
  description = "Displays the subnet names list."
}

output "sg-id" {
  value = azurerm_network_security_group.sg.id
  description = "Displays the security group id."
}

output "private-dns-zone-name" {
  value = azurerm_private_dns_zone.dns.name
  description = "Displays the private DNS zone name."
}