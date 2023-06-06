resource "azurerm_logic_app_workflow" "scale-up-workflow" {
  name                = "${var.prefix}-${var.cluster_name}-workflow-scale-up"
  resource_group_name = data.azurerm_resource_group.rg.name
  location            = data.azurerm_resource_group.rg.location
  tags                = merge(var.tags_map, {"weka_cluster": var.cluster_name})
  identity {
    type = "SystemAssigned"
  }
  access_control {
    trigger {
      allowed_caller_ip_address_range = [data.azurerm_subnet.subnets[0].address_prefix]
    }
    content {
      allowed_caller_ip_address_range = ["${local.my_ip}/32"]
    }
  }
  lifecycle {
    ignore_changes = [tags]
  }
  depends_on = [azurerm_linux_function_app.function_app,azurerm_linux_virtual_machine_scale_set.custom_image_vmss,azurerm_linux_virtual_machine_scale_set.default_image_vmss]
}

resource "azurerm_logic_app_trigger_recurrence" "scale-up-trigger" {
  name         = "run-every-day"
  logic_app_id = azurerm_logic_app_workflow.scale-up-workflow.id
  frequency    = "Minute"
  interval     = 1
  depends_on   = [azurerm_logic_app_workflow.scale-up-workflow]
}

resource "azurerm_logic_app_action_custom" "scale_up_logic_app_action_get_secret" {
  name         = "get-secret"
  logic_app_id = azurerm_logic_app_workflow.scale-up-workflow.id
  body = <<BODY
{
        "type": "Http",
        "inputs": {
          "uri": "https://${azurerm_key_vault.key_vault.name}.vault.azure.net/secrets/${azurerm_key_vault_secret.function_app_default_key.name}?api-version=2016-10-01",
          "method": "GET",
          "authentication": {
            "type": "ManagedServiceIdentity",
            "audience": "https://vault.azure.net"
          }
        },
        "runAfter": {},
        "runtimeConfiguration": {
          "secureData": {
            "properties": [
              "outputs"
            ]
          }
        }
}
BODY
  depends_on   = [azurerm_linux_function_app.function_app, azurerm_logic_app_workflow.scale-up-workflow]
}

resource "azurerm_logic_app_action_custom" "logic_app_action_scale_up" {
  name         = "scale-up"
  logic_app_id = azurerm_logic_app_workflow.scale-up-workflow.id
  body = <<BODY
{
    "inputs": {
        "retryPolicy": {
          "type": "none"
        },
        "body": {},
        "method": "POST",
        "headers": {
            "x-functions-key": "@body('${azurerm_logic_app_action_custom.scale_up_logic_app_action_get_secret.name}')?['value']"
        },
        "function": {
            "id": "${azurerm_linux_function_app.function_app.id}/functions/scale_up"
        }
    },
    "type": "Function",
     "runAfter": {
        "${azurerm_logic_app_action_custom.scale_up_logic_app_action_get_secret.name}": [
            "Succeeded"
      ]
  }
}
BODY
  depends_on   = [azurerm_linux_function_app.function_app, azurerm_logic_app_workflow.scale-up-workflow]
}

resource "azurerm_key_vault_access_policy" "scale-up-logic-app-get-secret-permission" {
  key_vault_id = azurerm_key_vault.key_vault.id
  tenant_id    = data.azurerm_client_config.current.tenant_id
  object_id    = azurerm_logic_app_workflow.scale-up-workflow.identity[0].principal_id
  secret_permissions = [
    "Get",
  ]
  depends_on = [azurerm_key_vault.key_vault,azurerm_logic_app_workflow.scale-up-workflow]
}

resource "azurerm_role_assignment" "scale-up-logic-app-key-vault-secrets-user" {
  scope                = azurerm_key_vault.key_vault.id
  role_definition_name = "Key Vault Secrets User"
  principal_id         = azurerm_logic_app_workflow.scale-up-workflow.identity[0].principal_id
  depends_on           = [azurerm_linux_function_app.function_app,azurerm_logic_app_workflow.scale-up-workflow]
}

resource "azurerm_role_assignment" "scale-up-logic-app-storage-account-contributor" {
  scope                = azurerm_storage_account.deployment_sa.id
  role_definition_name = "Storage Blob Data Contributor"
  principal_id         = azurerm_logic_app_workflow.scale-up-workflow.identity[0].principal_id
  depends_on           = [azurerm_logic_app_workflow.scale-up-workflow]
}

resource "azurerm_monitor_diagnostic_setting" "scale_up_logic_app_diagnostic_setting" {
  name                       = "${var.prefix}-${var.cluster_name}-workflow-scale-up-diagnostic"
  target_resource_id         = azurerm_logic_app_workflow.scale-up-workflow.id
  storage_account_id         = azurerm_storage_account.deployment_sa.id
  log_analytics_workspace_id = azurerm_log_analytics_workspace.la_workspace.id
  enabled_log {
    category = "WorkflowRuntime"

    retention_policy {
      enabled = false
    }
  }
  lifecycle {
    ignore_changes = [metric,log_analytics_destination_type]
  }
  depends_on = [azurerm_logic_app_workflow.scale-up-workflow]
}

resource "azurerm_logic_app_workflow" "scale-down-workflow" {
  name                = "${var.prefix}-${var.cluster_name}-workflow-scale-down"
  resource_group_name = data.azurerm_resource_group.rg.name
  location            = data.azurerm_resource_group.rg.location
  tags                = merge(var.tags_map, {"weka_cluster": var.cluster_name})

  identity {
    type = "SystemAssigned"
  }
  access_control {
    trigger {
      allowed_caller_ip_address_range = [data.azurerm_subnet.subnets[0].address_prefix]
    }
    content {
      allowed_caller_ip_address_range = ["${local.my_ip}/32"]
    }
  }
  lifecycle {
    ignore_changes = [tags]
  }
  depends_on = [azurerm_linux_function_app.function_app,azurerm_linux_virtual_machine_scale_set.custom_image_vmss,azurerm_linux_virtual_machine_scale_set.default_image_vmss]
}

resource "azurerm_logic_app_trigger_recurrence" "scale-down-trigger" {
  name         = "run-every-day"
  logic_app_id = azurerm_logic_app_workflow.scale-down-workflow.id
  frequency    = "Minute"
  interval     = 1
  depends_on   = [azurerm_logic_app_workflow.scale-down-workflow]
}

resource "azurerm_key_vault_access_policy" "logic-app-get-secret-permission" {
  key_vault_id = azurerm_key_vault.key_vault.id
  tenant_id    = data.azurerm_client_config.current.tenant_id
  object_id    = azurerm_logic_app_workflow.scale-down-workflow.identity[0].principal_id
  secret_permissions = [
    "Get",
  ]
  depends_on = [azurerm_key_vault.key_vault,azurerm_logic_app_workflow.scale-down-workflow]
}

resource "azurerm_role_assignment" "logic-app-key-vault-secrets-user" {
  scope                = azurerm_key_vault.key_vault.id
  role_definition_name = "Key Vault Secrets User"
  principal_id         = azurerm_logic_app_workflow.scale-down-workflow.identity[0].principal_id
  depends_on           = [azurerm_linux_function_app.function_app,azurerm_logic_app_workflow.scale-down-workflow]
}

resource "azurerm_role_assignment" "logic-app-storage-account-contributor" {
  scope                = azurerm_storage_account.deployment_sa.id
  role_definition_name = "Storage Blob Data Contributor"
  principal_id         = azurerm_logic_app_workflow.scale-down-workflow.identity[0].principal_id
  depends_on           = [azurerm_logic_app_workflow.scale-down-workflow]
}

resource "azurerm_logic_app_action_custom" "scale_down_logic_app_action_get_secret" {
  name         = "get-secret"
  logic_app_id = azurerm_logic_app_workflow.scale-down-workflow.id
  body = <<BODY
{
        "type": "Http",
        "inputs": {
          "uri": "https://${azurerm_key_vault.key_vault.name}.vault.azure.net/secrets/${azurerm_key_vault_secret.function_app_default_key.name}?api-version=2016-10-01",
          "method": "GET",
          "authentication": {
            "type": "ManagedServiceIdentity",
            "audience": "https://vault.azure.net"
          }
        },
        "runAfter": {},
        "runtimeConfiguration": {
          "secureData": {
            "properties": [
              "outputs"
            ]
          }
        }
}
BODY
  depends_on   = [azurerm_linux_function_app.function_app, azurerm_logic_app_workflow.scale-down-workflow]
}

resource "azurerm_logic_app_action_custom" "logic_app_action_fetch" {
  name         = "fetch"
  logic_app_id = azurerm_logic_app_workflow.scale-down-workflow.id
  body = <<BODY
{
    "inputs": {
        "retryPolicy": {
          "type": "none"
        },
        "body": {},
        "method": "POST",
        "headers": {
            "x-functions-key": "@body('${azurerm_logic_app_action_custom.scale_down_logic_app_action_get_secret.name}')?['value']"
        },
        "function": {
            "id": "${azurerm_linux_function_app.function_app.id}/functions/fetch"
        }
    },
    "type": "Function",
    "runtimeConfiguration": {
          "secureData": {
               "properties": [
                    "outputs"
              ]
          }
    },
    "runAfter": {
        "${azurerm_logic_app_action_custom.scale_down_logic_app_action_get_secret.name}": [
            "Succeeded"
          ]
     }
}
BODY
  depends_on   = [azurerm_linux_function_app.function_app, azurerm_logic_app_action_custom.scale_down_logic_app_action_get_secret]
}

resource "azurerm_logic_app_action_custom" "logic_app_action_scale_down" {
  name         = "scale-down"
  logic_app_id = azurerm_logic_app_workflow.scale-down-workflow.id
  body = <<BODY
{
    "inputs": {
        "retryPolicy": {
          "type": "none"
        },
        "body": "@body('${azurerm_logic_app_action_custom.logic_app_action_fetch.name}')",
        "method": "POST",
        "headers": {
            "x-functions-key": "@body('${azurerm_logic_app_action_custom.scale_down_logic_app_action_get_secret.name}')?['value']"
        },
        "function": {
            "id": "${azurerm_linux_function_app.function_app.id}/functions/scale_down"
        }
    },
    "type": "Function",
    "runtimeConfiguration": {
          "secureData": {
               "properties": [
                    "inputs"
              ]
          }
    },
     "runAfter": {
        "${azurerm_logic_app_action_custom.logic_app_action_fetch.name}": [
            "Succeeded"
      ]
  }
}
BODY
  depends_on   = [azurerm_linux_function_app.function_app,azurerm_logic_app_action_custom.logic_app_action_fetch]
}

resource "azurerm_logic_app_action_custom" "logic_app_action_terminate" {
  name         = "terminate"
  logic_app_id = azurerm_logic_app_workflow.scale-down-workflow.id
  body = <<BODY
{
    "inputs": {
        "retryPolicy": {
          "type": "none"
        },
        "body": "@body('${azurerm_logic_app_action_custom.logic_app_action_scale_down.name}')",
        "method": "POST",
        "headers": {
            "x-functions-key": "@body('${azurerm_logic_app_action_custom.scale_down_logic_app_action_get_secret.name}')?['value']"
        },
        "function": {
            "id": "${azurerm_linux_function_app.function_app.id}/functions/terminate"
        }
    },
    "type": "Function",
     "runAfter": {
        "${azurerm_logic_app_action_custom.logic_app_action_scale_down.name}": [
            "Succeeded"
      ]
  }
}
BODY
  depends_on   = [azurerm_linux_function_app.function_app,azurerm_logic_app_action_custom.logic_app_action_scale_down]
}

resource "azurerm_logic_app_action_custom" "logic_app_action_transient" {
  name         = "transient"
  logic_app_id = azurerm_logic_app_workflow.scale-down-workflow.id
  body = <<BODY
{
    "inputs": {
        "retryPolicy": {
          "type": "none"
        },
        "body": "@body('${azurerm_logic_app_action_custom.logic_app_action_terminate.name}')",
        "method": "POST",
        "headers": {
            "x-functions-key": "@body('${azurerm_logic_app_action_custom.scale_down_logic_app_action_get_secret.name}')?['value']"
        },
        "function": {
            "id": "${azurerm_linux_function_app.function_app.id}/functions/transient"
        }
    },
    "type": "Function",
     "runAfter": {
        "${azurerm_logic_app_action_custom.logic_app_action_terminate.name}": [
            "Succeeded"
      ]
  }
}
BODY
  depends_on = [azurerm_linux_function_app.function_app,azurerm_logic_app_action_custom.logic_app_action_terminate]
}

resource "azurerm_monitor_diagnostic_setting" "logic_app_diagnostic_setting" {
  name                       = "${var.prefix}-${var.cluster_name}-workflow-diagnostic"
  target_resource_id         = azurerm_logic_app_workflow.scale-down-workflow.id
  storage_account_id         = azurerm_storage_account.deployment_sa.id
  log_analytics_workspace_id = azurerm_log_analytics_workspace.la_workspace.id
  enabled_log {
    category = "WorkflowRuntime"

    retention_policy {
      enabled = false
    }
  }
  lifecycle {
    ignore_changes = [metric,log_analytics_destination_type]
  }
  depends_on = [azurerm_logic_app_workflow.scale-down-workflow]
}
