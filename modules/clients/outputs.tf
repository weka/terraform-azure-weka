output "client-name" {
  value = azurerm_linux_virtual_machine_scale_set.vmss.name
}

output "backend" {
  value = var.backend_ips
}