locals {
  vmss_name            = "${var.prefix}-${var.cluster_name}-vmss"
  clients_vmss_name    = var.clients_number > 0 && var.clients_use_vmss ? module.clients[0].vmss_name : ""
  key_vault_name       = azurerm_key_vault.key_vault.name
  vm_ips               = local.assign_public_ip ? "az vmss list-instance-public-ips -g ${var.rg_name} --name ${local.vmss_name} --subscription ${var.subscription_id} --query \"[].ipAddress\" \n" : "az vmss nic list -g ${var.rg_name} --vmss-name ${local.vmss_name} --subscription ${var.subscription_id} --query \"[].ipConfigurations[]\" | jq -r '.[] | select(.name==\"ipconfig0\")'.privateIPAddress \n"
  clients_vmss_ips     = local.assign_public_ip ? "az vmss list-instance-public-ips -g ${var.rg_name} --name ${local.clients_vmss_name} --subscription ${var.subscription_id} --query \"[].ipAddress\" \n" : "az vmss nic list -g ${var.rg_name} --vmss-name ${local.clients_vmss_name} --subscription ${var.subscription_id} --query \"[].ipConfigurations[]\" | jq -r '.[] | select(.name==\"ipconfig0\")'.privateIPAddress \n"
  ssh_keys_commands    = "########################################## Download ssh keys command from blob ###########################################################\n az keyvault secret download --file private.pem --encoding utf-8 --vault-name  ${local.key_vault_name} --name private-key --query \"value\" \n az keyvault secret download --file public.pub --encoding utf-8 --vault-name  ${local.key_vault_name} --name public-key --query \"value\"\n"
  blob_commands        = var.ssh_public_key == null ? local.ssh_keys_commands : ""
  private_ssh_key_path = var.ssh_public_key == null ? local.ssh_private_key_path : null
  resource_group_name  = data.azurerm_resource_group.rg.name
  functions_url = {
    progressing_status = {
      url  = "https://${local.function_app_name}.azurewebsites.net/api/status"
      body = { "type" : "progress" }
    }
    status = {
      url  = "https://${local.function_app_name}.azurewebsites.net/api/status"
      body = { "type" : "status" }
    }
    resize = {
      uri  = "https://${local.function_app_name}.azurewebsites.net/api/resize"
      body = { "value" : 7 }
    }
  }
}

output "backend_lb_private_ip" {
  value       = var.create_lb ? azurerm_lb.backend_lb[0].private_ip_address : null
  description = "Backend load balancer ip address"
}

output "functions_url" {
  value       = local.functions_url
  description = "Functions url and body for api request"
}

output "vmss_name" {
  value = local.vmss_name
}

output "function_app_name" {
  value       = local.function_app_name
  description = "Function app name"
}

output "function_key_name" {
  value       = "functionKeys"
  description = "Function app key name"
}

output "vm_username" {
  value       = var.vm_username
  description = "Provided as part of output for automated use of terraform, ssh user to weka cluster vm"
}

output "backend_ips" {
  value       = local.vm_ips
  description = "If 'assign_public_ip' is set to true, it will output the public ips, If no it will output the private ips"
}

output "clients_vmss_name" {
  value = local.clients_vmss_name != "" ? local.clients_vmss_name : null
}

output "client_ips" {
  value       = var.clients_number > 0 && !var.clients_use_vmss ? module.clients[0].client_ips : null
  description = "If 'private_network' is set to false, it will output clients public ips, otherwise private ips."
}

output "client_vmss_ips" {
  value       = var.clients_number > 0 && var.clients_use_vmss ? local.clients_vmss_ips : null
  description = "If 'private_network' is set to false, it will output clients public ips, otherwise private ips."
}

output "nfs_vmss_name" {
  value       = var.nfs_protocol_gateways_number > 0 ? module.nfs_protocol_gateways[0].nfs_vmss_name : null
  description = "NFS protocol gateway vmss name"
}

output "smb_protocol_gateway_ips" {
  value       = var.smb_protocol_gateways_number > 0 ? module.smb_protocol_gateways[0].protocol_gateways_ips : null
  description = "If 'private_network' is set to false, it will output smb protocol gateway public ips, otherwise private ips."
}

output "s3_protocol_gateway_ips" {
  value       = var.s3_protocol_gateways_number > 0 ? module.s3_protocol_gateways[0].protocol_gateways_ips : null
  description = "If 'private_network' is set to false, it will output smb protocol gateway public ips, otherwise private ips."
}

output "private_ssh_key" {
  value       = local.private_ssh_key_path
  description = "If 'ssh_public_key' is set to null and no file provided, it will output the private ssh key location."
}

output "key_vault_name" {
  value       = local.key_vault_name
  description = "Keyault name"
}

output "subnet_name" {
  value       = local.subnet_name
  description = "Subnet name"
}

output "vnet_name" {
  value       = local.vnet_name
  description = "Virtual network name"
}

output "vnet_rg_name" {
  value       = local.vnet_rg_name
  description = "Virtual network resource group name"
}

output "sg_id" {
  value       = local.sg_id
  description = "Security group id"
}

output "ppg_id" {
  value       = local.placement_group_id
  description = "Placement proximity group id"
}

output "weka_cluster_admin_password_secret_name" {
  value       = azurerm_key_vault_secret.weka_password_secret.name
  description = "Weka cluster admin password secret name"
}


locals {
  resize_helper_command   = var.storage_account_public_network_access != "Enabled" ? "" : <<EOT
########################################## Resize cluster #################################################################################
function_key=$(az functionapp keys list --name ${local.function_app_name} --resource-group ${local.resource_group_name} --subscription ${var.subscription_id} --query functionKeys -o tsv)
curl --fail https://${local.function_app_name}.azurewebsites.net/api/resize?code=$function_key -H "Content-Type:application/json" -d '{"value":ENTER_NEW_VALUE_HERE}'
EOT
  scale_up_helper_command = var.storage_account_public_network_access == "Enabled" ? "" : <<EOT
########################################## Scale up cluster #################################################################################
function_key=$(az functionapp keys list --name ${local.function_app_name} --resource-group ${local.resource_group_name} --subscription ${var.subscription_id} --query functionKeys -o tsv)
curl --fail https://${local.function_app_name}.azurewebsites.net/api/scale_up?code=$function_key
EOT
}

output "cluster_helper_commands" {
  value       = <<EOT
########################################## Get function key #####################################################################
az functionapp keys list --name ${local.function_app_name} --resource-group ${local.resource_group_name} --subscription ${var.subscription_id} --query functionKeys -o tsv

########################################## Get clusterization status #####################################################################
function_key=$(az functionapp keys list --name ${local.function_app_name} --resource-group ${local.resource_group_name} --subscription ${var.subscription_id} --query functionKeys -o tsv)
curl --fail https://${local.function_app_name}.azurewebsites.net/api/status?code=$function_key -H "Content-Type:application/json" -d '{"type": "progress"}'

########################################## Get cluster status ############################################################################
function_key=$(az functionapp keys list --name ${local.function_app_name} --resource-group ${local.resource_group_name} --subscription ${var.subscription_id} --query functionKeys -o tsv)
curl --fail https://${local.function_app_name}.azurewebsites.net/api/status?code=$function_key

######################################### Fetch weka cluster password ####################################################################
az keyvault secret show --vault-name ${local.key_vault_name} --name ${azurerm_key_vault_secret.weka_password_secret.name} | jq .value

${local.blob_commands}

${local.resize_helper_command}
${local.scale_up_helper_command}

########################################## pre-terraform destroy, cluster terminate function ################
backends_vmss_name=${local.vmss_name}
az vmss delete --name <ENTER YOUR BACKENDS VMSS_NAME HERE> --resource-group ${var.rg_name} --force-deletion true --subscription ${var.subscription_id}

EOT
  description = "Useful commands and script to interact with weka cluster"
}
