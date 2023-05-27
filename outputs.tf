locals {
  vm_ips            = var.private_network ? "az vmss nic list -g ${var.rg_name} --vmss-name ${azurerm_linux_virtual_machine_scale_set.vmss.name} --subscription ${var.subscription_id} --query \"[].ipConfigurations[]\" | jq -r '.[] | select(.name==\"ipconfig0\")'.privateIPAddress" : "az vmss list-instance-public-ips -g ${var.rg_name} --name ${azurerm_linux_virtual_machine_scale_set.vmss.name} --subscription ${var.subscription_id} --query \"[].ipAddress\" \n"
  ssh_keys_commands = "########################################## Download ssh keys command from blob ###########################################################\n az keyvault secret download --file private.pem --encoding utf-8 --vault-name  ${azurerm_key_vault.key_vault.name} --name private-key --query \"value\" \n az keyvault secret download --file public.pub --encoding utf-8 --vault-name  ${azurerm_key_vault.key_vault.name} --name public-key --query \"value\"\n"
  blob_commands     = var.ssh_public_key == null ? local.ssh_keys_commands : ""
  path_ssh_keys     = var.ssh_public_key == null ? "${local.ssh_path}-public-key.pub \n ${local.ssh_path}-private-key.pem" : "${var.ssh_private_key} \n ${var.ssh_public_key}"
  mngmt_vm_public_ip   = azurerm_linux_virtual_machine.management_vm.public_ip_address
  mngmt_vm_private_ip  = azurerm_linux_virtual_machine.management_vm.private_ip_address
  function_url         = "${local.mngmt_vm_private_ip}:${var.http_server_port}"
}
output "cluster_helpers_commands" {
  value       = <<EOT
######################################## Connect to management machine ###################################################################
ssh -i ${local.path_ssh_private_key} ${var.vm_username}@${local.mngmt_vm_public_ip}

########################################## Get clusterization status #####################################################################
curl --fail ${local.function_url}/status -H "Content-Type:application/json" -d '{"type": "progress"}'

########################################## Get cluster status ############################################################################
curl --fail ${local.function_url}/status

######################################### Fetch weka cluster password ####################################################################
az keyvault secret show --vault-name ${azurerm_key_vault.key_vault.name} --name weka-password | jq .value

${local.blob_commands}
############################################## Path to ssh keys  ##########################################################################
${local.path_ssh_private_key}
${local.path_ssh_public_key}

################################################ Vms ips ##################################################################################
${local.vm_ips}
username: ${var.vm_username}

########################################## Resize cluster #################################################################################
curl --fail ${local.function_url}/resize -H "Content-Type:application/json" -d '{"value":ENTER_NEW_VALUE_HERE}'
EOT
  description = "Useful commands and script to interact with weka cluster"
}
