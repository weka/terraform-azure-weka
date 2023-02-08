output "ssh-key-files-path" {
  value = var.ssh_public_key == null ? "${local.ssh_path}-public-key.pub, ${local.ssh_path}-private-key.pem" : "${var.ssh_public_key} , ${var.ssh_private_key}"
  description = "Displays the path of the ssh keys."
}

locals {
  key_vault_name = var.ssh_public_key == null ? azurerm_key_vault.key_vault.name : ""
  blob_commands =<<EOT
########################################## download ssh keys command from blob ###########################################################
  CLUSTER: ${var.cluster_name}
  az keyvault secret download --file private.pem --encoding utf-8 --vault-name  ${local.key_vault_name} --name private-key --query "value"
  az keyvault secret download --file public.pub --encoding utf-8 --vault-name  ${local.key_vault_name} --name public-key --query "value"
EOT
}

output "ssh-key-download-blob" {
  value = var.ssh_public_key == null ? local.blob_commands : "No private ssh file created"
  description = "Commands to download the ssh keys from the Azure Blob."
}

output "get-cluster-status" {
  value =<<EOT
########################################## get cluster status #############################################################################
function_key=$(az functionapp keys list --name ${azurerm_linux_function_app.function_app.name} --resource-group ${data.azurerm_resource_group.rg.name} --subscription ${var.subscription_id} --query functionKeys -o tsv)
curl https://${var.prefix}-${var.cluster_name}-function-app.azurewebsites.net/api/status?code=$function_key
EOT
  description = "A command to get the cluster status."
}

output "get-vms-private-ips" {
  value =<<EOT
################################## get private ips ######################################################
  az vmss nic list -g ${var.rg_name} --vmss-name ${azurerm_linux_virtual_machine_scale_set.vmss.name} --subscription ${var.subscription_id} --query "[].ipConfigurations[].privateIpAddress"
EOT
  description = "Get private ips of vms"
}

output "get-vms-public-ips" {
  value =<<EOT
################################## get public ips ######################################################
  az vmss list-instance-public-ips -g ${var.rg_name} --name ${azurerm_linux_virtual_machine_scale_set.vmss.name} --subscription ${var.subscription_id} --query "[].ipAddress"
EOT
  description = "Get public ips of vms"
}

output "fetch-weka-cluster-password" {
  value =<<EOT
################################## fetch weka cluster password ######################################################
  az keyvault secret show --vault-name ${azurerm_key_vault.key_vault.name} --name weka-password | jq .value
EOT
  description = "Fetch weka cluster password"
}
