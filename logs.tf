resource "azurerm_log_analytics_workspace" "la_workspace" {
  name                = "${local.alphanumeric_prefix_name}-${local.alphanumeric_cluster_name}-workspace"
  location            = data.azurerm_resource_group.rg.location
  resource_group_name = data.azurerm_resource_group.rg.name
  sku                 = "PerGB2018"
  retention_in_days   = 30
  lifecycle {
    ignore_changes = [tags]
  }
}

resource "azurerm_application_insights" "application_insights" {
  name                = "${var.prefix}-${var.cluster_name}-application-insights"
  location            = data.azurerm_resource_group.rg.location
  resource_group_name = data.azurerm_resource_group.rg.name
  workspace_id        = azurerm_log_analytics_workspace.la_workspace.id
  application_type    = "web"
  lifecycle {
    ignore_changes = [tags]
  }
}

resource "azurerm_monitor_diagnostic_setting" "insights_diagnostic_setting" {
  name                       = "${var.prefix}-${var.cluster_name}-insights-diagnostic-setting"
  target_resource_id         = azurerm_application_insights.application_insights.id
  storage_account_id         = azurerm_storage_account.deployment_sa.id
  log_analytics_workspace_id = azurerm_log_analytics_workspace.la_workspace.id
  enabled_log {
    category = "AppTraces"

    retention_policy {
      enabled = false
    }
  }
  lifecycle {
    ignore_changes = [metric, log_analytics_destination_type]
  }
  depends_on = [azurerm_log_analytics_workspace.la_workspace]
}
