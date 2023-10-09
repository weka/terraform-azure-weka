output "service_principal_name" {
  description = "Service principal name"
  value       = azuread_service_principal.main.display_name
}

output "service_principal_object_id" {
  description = "Object id of service principal"
  value       = azuread_service_principal.main.object_id
}

output "client_id" {
  description = "The application id of AzureAD application created."
  value       = azuread_application.main.application_id
}

output "client_secret" {
  description = "Password for service principal."
  value       = azuread_service_principal_password.main.value
  sensitive   = true
}
