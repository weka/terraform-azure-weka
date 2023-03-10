locals {
  vmss_name         = var.custom_image_id != null ? azurerm_orchestrated_virtual_machine_scale_set.custom_image_vmss[0].name : azurerm_orchestrated_virtual_machine_scale_set.default_image_vmss[0].name
  ssh_keys_commands = "########################################## Download ssh keys command from blob ###########################################################\n az keyvault secret download --file private.pem --encoding utf-8 --vault-name  ${azurerm_key_vault.key_vault.name} --name private-key --query \"value\" \n az keyvault secret download --file public.pub --encoding utf-8 --vault-name  ${azurerm_key_vault.key_vault.name} --name public-key --query \"value\"\n"
  blob_commands     = var.ssh_public_key == null ? local.ssh_keys_commands : ""
  path_ssh_keys     = var.ssh_public_key == null ? "${local.ssh_path}-public-key.pub \n ${local.ssh_path}-private-key.pem" : "${var.ssh_private_key} \n ${var.ssh_public_key}"
}
output "cluster_helpers_commands" {
  value = <<EOT
########################################## Get cluster status ############################################################################
function_key=$(az functionapp keys list --name ${azurerm_linux_function_app.function_app.name} --resource-group ${data.azurerm_resource_group.rg.name} --subscription ${var.subscription_id} --query functionKeys -o tsv)
curl https://${var.prefix}-${var.cluster_name}-function-app.azurewebsites.net/api/status?code=$function_key

######################################### Fetch weka cluster password ####################################################################
az keyvault secret show --vault-name ${azurerm_key_vault.key_vault.name} --name weka-password | jq .value

${local.blob_commands}
############################################## Path to ssh keys  ##########################################################################
${local.path_ssh_keys}

################################################ Vms ips ##################################################################################
az vm list-ip-addresses --ids $(az vm list --subscription ${var.subscription_id} -g ${var.rg_name} --query "[? contains(name, '${local.vmss_name}')].id" -o tsv) --query "[]
.virtualMachine.{Name:name,PrivateIp:network.privateIpAddresses[0],PublicIp:network.publicIpAddresses[0].ipAddress}" -o table

########################################## Resize cluster #################################################################################
function_key=$(az functionapp keys list --name ${azurerm_linux_function_app.function_app.name} --resource-group ${data.azurerm_resource_group.rg.name} --subscription ${var.subscription_id} --query functionKeys -o tsv)
curl https://${var.prefix}-${var.cluster_name}-function-app.azurewebsites.net/api/resize?code=$function_key -H "Content-Type:application/json" -d '{"value":ENTER_NEW_VALUE_HERE}'
EOT
  description = "Useful commands and script to interact with weka cluster"
}

