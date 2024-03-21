output "protocol_gateways_ips" {
  value       = var.assign_public_ip ? azurerm_linux_virtual_machine.this[*].public_ip_address : local.first_nic_private_ips
  description = "If 'assign_public_ip' is set to true, it will output backends public ips, otherwise private ips."
}

output "nfs_vmss_name" {
  value       = var.protocol == "NFS" ? azurerm_orchestrated_virtual_machine_scale_set.nfs[0].name : null
  description = "The name of the NFS VMSS."
}
