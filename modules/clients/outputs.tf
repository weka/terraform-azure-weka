output "client_ips" {
  value = var.assign_public_ip ? azurerm_linux_virtual_machine.this[*].public_ip_address : azurerm_linux_virtual_machine.this[*].private_ip_address
}

output "client_names" {
  value = azurerm_linux_virtual_machine.this[*].name
}
