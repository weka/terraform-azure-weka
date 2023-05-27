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

resource "azurerm_monitor_data_collection_rule" "dcr" {
  name                = "${var.prefix}-${var.cluster_name}-dcr"
  resource_group_name = data.azurerm_resource_group.rg.name
  location            = data.azurerm_resource_group.rg.location
  description         = "Syslog collection rule for VMs"

  destinations {
    log_analytics {
      workspace_resource_id = azurerm_log_analytics_workspace.la_workspace.id
      name                  = "la_workspace"
    }
  }
  data_sources {
    syslog {
      streams        = ["Microsoft-Syslog"]
      facility_names = ["*"]
      log_levels     = ["Info"]
      name           = "${var.prefix}-${var.cluster_name}-vms-syslog"
    }
  }
  data_flow {
    streams      = ["Microsoft-Syslog"]
    destinations = ["la_workspace"]
  }
}

resource "azurerm_monitor_data_collection_rule_association" "mngmt_vm_syslog" {
  name                    = "mngmt-vm-syslog-dcr"
  target_resource_id      = azurerm_linux_virtual_machine.management_vm.id
  data_collection_rule_id = azurerm_monitor_data_collection_rule.dcr.id
}

resource "azurerm_monitor_data_collection_rule_association" "vmss_syslog" {
  name                    = "vmss-syslog-dcr"
  target_resource_id      = var.custom_image_id != null ? azurerm_linux_virtual_machine_scale_set.custom_image_vmss.0.id : azurerm_linux_virtual_machine_scale_set.default_image_vmss.0.id
  data_collection_rule_id = azurerm_monitor_data_collection_rule.dcr.id
}
