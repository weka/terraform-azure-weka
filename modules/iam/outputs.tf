output "logic_app_identity_id" {
  value       = var.logic_app_identity_name == "" ? azurerm_user_assigned_identity.logic_app[0].id : data.azurerm_user_assigned_identity.logic_app[0].id
  description = "The ID of the managed identity for the logic app"
}

output "logic_app_identity_principal_id" {
  value       = var.logic_app_identity_name == "" ? azurerm_user_assigned_identity.logic_app[0].principal_id : data.azurerm_user_assigned_identity.logic_app[0].principal_id
  description = "The principal ID of the managed identity for the logic app"
}

output "function_app_identity_id" {
  value       = var.function_app_identity_name == "" ? azurerm_user_assigned_identity.function_app[0].id : data.azurerm_user_assigned_identity.function_app[0].id
  description = "The ID of the managed identity for the function app"
}

output "function_app_identity_principal_id" {
  value       = var.function_app_identity_name == "" ? azurerm_user_assigned_identity.function_app[0].principal_id : data.azurerm_user_assigned_identity.function_app[0].principal_id
  description = "The principal ID of the managed identity for the function app"
}

output "function_app_identity_client_id" {
  value       = var.function_app_identity_name == "" ? azurerm_user_assigned_identity.function_app[0].client_id : data.azurerm_user_assigned_identity.function_app[0].client_id
  description = "The client ID of the managed identity for the function app"
}

output "vmss_identity_id" {
  value       = var.vmss_identity_name == "" ? azurerm_user_assigned_identity.vmss[0].id : data.azurerm_user_assigned_identity.vmss[0].id
  description = "The ID of the managed identity for the vmss"
}
