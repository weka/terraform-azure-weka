locals {
  key_vault_name = azurerm_key_vault.key_vault.name
  vm_ips            = var.private_network ? "az vmss nic list -g ${var.rg_name} --vmss-name ${azurerm_linux_virtual_machine_scale_set.vmss.name} --subscription ${var.subscription_id} --query \"[].ipConfigurations[]\" | jq -r '.[] | select(.name==\"ipconfig0\")'.privateIPAddress" : "az vmss list-instance-public-ips -g ${var.rg_name} --name ${azurerm_linux_virtual_machine_scale_set.vmss.name} --subscription ${var.subscription_id} --query \"[].ipAddress\" \n"
  clients_ips       = var.clients_number > 0 ? var.private_network ? "az vmss nic list -g ${var.rg_name} --vmss-name ${module.clients[0].client-name} --subscription ${var.subscription_id} --query \"[].ipConfigurations[]\" | jq -r '.[] | select(.name==\"ipconfig0\")'.privateIPAddress" : "az vmss list-instance-public-ips -g ${var.rg_name} --name ${module.clients[0].client-name} --subscription ${var.subscription_id} --query \"[].ipAddress\" \n" : ""
  protocol_gw_ips   = var.protocol_gateways_number > 0 ? var.private_network ? "az vmss nic list -g ${var.rg_name} --vmss-name ${module.protocol_gateways[0].vmss_name} --subscription ${var.subscription_id} --query \"[].ipConfigurations[]\" | jq -r '.[] | select(.name==\"ipconfig0\")'.privateIPAddress" : "az vmss list-instance-public-ips -g ${var.rg_name} --name ${module.protocol_gateways[0].vmss_name} --subscription ${var.subscription_id} --query \"[].ipAddress\" \n" : ""
  ssh_keys_commands = "########################################## Download ssh keys command from blob ###########################################################\n az keyvault secret download --file private.pem --encoding utf-8 --vault-name  ${local.key_vault_name} --name private-key --query \"value\" \n az keyvault secret download --file public.pub --encoding utf-8 --vault-name  ${local.key_vault_name} --name public-key --query \"value\"\n"
  blob_commands     = var.ssh_public_key == null ? local.ssh_keys_commands : ""
  private_ssh_key_path     = var.ssh_public_key == null ? "${local.ssh_path}-private-key.pem" : null
  resource_group_name = data.azurerm_resource_group.rg.name
}

output "function_app_name" {
  value = local.function_app_name
}

output "resource_group_name" {
  value = local.resource_group_name
}

output "subscription_id" {
  value = var.subscription_id
}

output "prefix" {
  value = var.prefix
}

output "cluster_name" {
  value = var.cluster_name
}

output "key_vault_name" {
  value = local.key_vault_name
}

output "ssh_user" {
  value = var.vm_username
  description = "ssh user for weka cluster"
}

output "backend_ips" {
  value = local.vm_ips
}

output "client_ips" {
  value = local.clients_ips
  description = "If 'private_network' is set to false, it will output clients public ips, otherwise private ips."
}

output "protocol_gateway_ips" {
  value = local.protocol_gw_ips
  description = "If 'private_network' is set to false, it will output protocol gateway public ips, otherwise private ips."
}

output "private_ssh_key" {
  value = local.private_ssh_key_path
}

output "cluster_helper_commands" {
  value = <<EOT
########################################## Get clusterization status #####################################################################
function_key=$(az functionapp keys list --name ${local.function_app_name} --resource-group ${local.resource_group_name} --subscription ${var.subscription_id} --query functionKeys -o tsv)
curl --fail https://${local.function_app_name}.azurewebsites.net/api/status?code=$function_key -H "Content-Type:application/json" -d '{"type": "progress"}'

########################################## Get cluster status ############################################################################
function_key=$(az functionapp keys list --name ${local.function_app_name} --resource-group ${data.azurerm_resource_group.rg.name} --subscription ${var.subscription_id} --query functionKeys -o tsv)
curl --fail https://${local.function_app_name}.azurewebsites.net/api/status?code=$function_key

######################################### Fetch weka cluster password ####################################################################
az keyvault secret show --vault-name ${local.key_vault_name} --name weka-password | jq .value

${local.blob_commands}

########################################## Resize cluster #################################################################################
function_key=$(az functionapp keys list --name ${local.function_app_name} --resource-group ${data.azurerm_resource_group.rg.name} --subscription ${var.subscription_id} --query functionKeys -o tsv)
curl --fail https://${local.function_app_name}.azurewebsites.net/api/resize?code=$function_key -H "Content-Type:application/json" -d '{"value":ENTER_NEW_VALUE_HERE}'

EOT
  description = "Useful commands and script to interact with weka cluster"
}
