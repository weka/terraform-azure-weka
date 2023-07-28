output "vmss_name" {
  value       = azurerm_linux_virtual_machine_scale_set.vmss.name
  description = "Protocol gateway vmss name"
}

output "vmss_principal_id" {
  value       = azurerm_linux_virtual_machine_scale_set.vmss.identity[0].principal_id
  description = "Protocol gateway vmss principal id"
}
