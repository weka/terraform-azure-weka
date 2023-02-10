provider "azurerm" {
  client_id       = var.client_id
  tenant_id       = var.tenant_id
  client_secret   = var.client_secret

  features {
  }
}

