output "protocol_gateways_ips" {
  value       = var.assign_public_ip ? azurerm_linux_virtual_machine.this.*.public_ip_address : local.first_nic_private_ips
  description = "If 'assign_public_ip' is set to true, it will output backends public ips, otherwise private ips."
}