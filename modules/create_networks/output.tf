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
  value = var.create_private_dns_zone ? azurerm_private_dns_zone.dns[0].name : null
  description = "Displays the private DNS zone name."
}

output "vnet_rg_name" {
  value = var.vnet_rg_name == null ? var.rg_name : var.vnet_rg_name
  description = "Resource group name of vnet."
}

output "sites-dns-zone-name" {
  value = var.sites_dns_zone_name == null ? azurerm_private_dns_zone.sites_dns[0].name : var.sites_dns_zone_name
}

output "blob-dns-zone-name" {
  value = var.private_dns_blob_name == null ? azurerm_private_dns_zone.blob-dns-zone[0].name : var.private_dns_blob_name
}

output "subnets_delegation_names" {
  value = length(var.subnets_delegation_names) == 0 ? azurerm_subnet.subnet-delegation.*.name : var.subnets_delegation_names
}

output "keyvault-dns-zone-name" {
  value = var.keyvault_dns_zone_name == null ? azurerm_private_dns_zone.keyvault-dns-zone[0].name : var.keyvault_dns_zone_name
}