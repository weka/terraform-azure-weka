resource "azurerm_storage_account" "sa" {
  count                    = var.create_ml ? 1 : 0
  name                     = "${var.prefix}mlsa"
  location                 = data.azurerm_resource_group.rg.location
  resource_group_name      = var.rg_name
  account_tier             = "Standard"
  account_replication_type = "GRS"
  lifecycle {
    ignore_changes = all
  }
}

resource "azurerm_application_insights" "insights" {
  count               = var.create_ml ? 1 : 0
  name                = "${var.prefix}-workspace-insights"
  location            = data.azurerm_resource_group.rg.location
  resource_group_name = var.rg_name
  application_type    = "web"
}

data "azurerm_key_vault" "vault" {
  name                = var.key_vault_name
  resource_group_name = var.rg_name
}

resource "azurerm_machine_learning_workspace" "ml" {
  count                         = var.create_ml ? 1 : 0
  name                          = "${var.prefix}-workspace-ml"
  location                      = data.azurerm_resource_group.rg.location
  resource_group_name           = var.rg_name
  application_insights_id       = azurerm_application_insights.insights[0].id
  key_vault_id                  = data.azurerm_key_vault.vault.id
  storage_account_id            = azurerm_storage_account.sa[0].id
  public_network_access_enabled = true

  identity {
    type = "SystemAssigned"
  }
  lifecycle {
    ignore_changes = all
  }
}
