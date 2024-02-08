locals {
  vmss_name = var.use_vmss ? azurerm_linux_virtual_machine_scale_set.vmss[0].name : null
  vmss_ips  = var.use_vmss ? var.assign_public_ip ? "az vmss list-instance-public-ips -g ${var.rg_name} --name ${local.vmss_name} --subscription ${var.subscription_id} --query \"[].ipAddress\" \n" : "az vmss nic list -g ${var.rg_name} --vmss-name ${local.vmss_name} --subscription ${var.subscription_id} --query \"[].ipConfigurations[]\" | jq -r '.[] | select(.name==\"ipconfig0\")'.privateIPAddress \n" : null
  vms_ips   = !var.use_vmss ? var.assign_public_ip ? azurerm_linux_virtual_machine.this[*].public_ip_address : azurerm_linux_virtual_machine.this[*].private_ip_address : null
}

output "clients_ips" {
  value = var.use_vmss ? [local.vmss_ips] : local.vms_ips
}

output "client_names" {
  value = var.use_vmss ? [local.vmss_name] : azurerm_linux_virtual_machine.this[*].name
}
