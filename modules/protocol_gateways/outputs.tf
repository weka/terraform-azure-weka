output "vmss_name" {
  value       = azurerm_linux_virtual_machine_scale_set.vmss.name
  description = "Protocol gateway vmss name"
}