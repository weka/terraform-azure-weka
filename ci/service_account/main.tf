provider "azurerm" {
  features {}
}

data "azuread_client_config" "current" {}


resource "azuread_application" "main" {
  display_name = var.service_principal_name
  owners       = [data.azuread_client_config.current.object_id]
}

resource "azuread_service_principal" "main" {
  application_id = azuread_application.main.application_id
  owners         = [data.azuread_client_config.current.object_id]
  description    = var.description
}

resource "azuread_directory_role" "main" {
  display_name = "Application Administrator"
}

resource "azuread_directory_role_assignment" "main" {
  role_id             = azuread_directory_role.main.template_id
  principal_object_id = azuread_service_principal.main.object_id
}

resource "azuread_service_principal_password" "main" {
  service_principal_id = azuread_service_principal.main.object_id
}

resource "azurerm_role_assignment" "main" {
  count                = length(var.assignments)
  name                 = var.azure_role_name
  description          = var.azure_role_description
  scope                = var.assignments[count.index].scope
  role_definition_name = var.assignments[count.index].role_definition_name
  principal_id         = azuread_service_principal.main.object_id
}
