output "vnet_name" {
  value       = var.vnet_name == "" ? azurerm_virtual_network.vnet[0].name : var.vnet_name
  description = "Displays the VNet name."
}

output "subnet_name" {
  value       = var.subnet_name == "" ? azurerm_subnet.subnet[0].name : var.subnet_name
  description = "Displays the subnet name list."
}

output "sg_id" {
  value       = var.sg_id == "" ? azurerm_network_security_group.sg[0].id : var.sg_id
  description = "Displays the security group id."
}

output "private_dns_zone_name" {
  value       = var.private_dns_zone_name == "" ? azurerm_private_dns_zone.dns[0].name : var.private_dns_zone_name
  description = "Displays the private DNS zone name."
}

output "vnet_rg_name" {
  value       = var.vnet_rg_name == "" ? var.rg_name : var.vnet_rg_name
  description = "Resource group name of vnet."
}

output "private_dns_rg_name" {
  value       = var.private_dns_rg_name == "" ? var.rg_name : var.private_dns_rg_name
  description = "The private DNS zone resource group name."
}
