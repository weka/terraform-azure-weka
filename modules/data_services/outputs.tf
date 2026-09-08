output "data_services_name" {
  value       = var.data_services_name
  description = "Data services name"
}

output "data_services_vmss_name" {
  value       = azurerm_linux_virtual_machine_scale_set.this.name
  description = "The name of the data services VMSS."
}
