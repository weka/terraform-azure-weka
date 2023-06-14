resource "azurerm_resource_group_template_deployment" "api_connections_template_deployment" {
  name                = "${var.prefix}-${var.cluster_name}-keyvault-api-connection-deploy"
  resource_group_name = data.azurerm_resource_group.rg.name
  deployment_mode     = "Incremental"
  template_content    = <<TEMPLATE
{
    "$schema": "https://schema.management.azure.com/schemas/2019-04-01/deploymentTemplate.json#",
    "contentVersion": "1.0.0.0",
    "parameters": {
        "connections_keyvault_name": {
            "defaultValue": "[concat('${azurerm_key_vault.key_vault.name}', '-connection')]",
            "type": "string"
        }
    },
    "variables": {},
    "resources": [
   {
    "type": "Microsoft.Web/connections",
    "apiVersion": "2016-06-01",
    "name": "[parameters('connections_keyvault_name')]",
    "location": "[resourceGroup().location]",
    "kind": "V1",
    "properties": {
        "displayName": "[concat('${azurerm_key_vault.key_vault.name}', '-connection')]",
        "statuses": [
            {
                "status": "Connected"
            }
        ],
        "parameterValueType": "Alternative",
        "alternativeParameterValues": {
          "vaultName": "${azurerm_key_vault.key_vault.name}"
        },
        "api": {
            "name": "[parameters('connections_keyvault_name')]",
            "displayName": "Azure Key Vault",
            "description": "Azure Key Vault is a service to securely store and access secrets.",
            "id": "/subscriptions/${var.subscription_id}/providers/Microsoft.Web/locations/${data.azurerm_resource_group.rg.location}/managedApis/keyvault",
            "type": "Microsoft.Web/locations/managedApis"
        },
        "testLinks": [
          {
            "requestUri": "[concat('https://management.azure.com:443/subscriptions/${var.subscription_id}/resourceGroups/${data.azurerm_resource_group.rg.name}/providers/Microsoft.Web/connections/', parameters('connections_keyvault_name'), '/extensions/proxy/testconnection?api-version=2016-06-01')]",
            "method": "get"
            }
          ]
      }
     }
    ],
    "outputs": {
        "keyvaultid":{
            "type": "string",
            "value" : "[resourceId('Microsoft.Web/connections', parameters('connections_keyvault_name'))]"
        }
    }
}
TEMPLATE
  depends_on = [azurerm_key_vault.key_vault, azurerm_linux_virtual_machine_scale_set.vmss]
  lifecycle {
    ignore_changes = [template_content]
  }
}

resource "azurerm_resource_group_template_deployment" "workflow_scale_down_template_deployment" {
  name                = "${var.prefix}-${var.cluster_name}-workflow-scale-down-deploy"
  resource_group_name = data.azurerm_resource_group.rg.name
  deployment_mode     = "Incremental"
  template_content    = <<TEMPLATE
  {
    "$schema": "https://schema.management.azure.com/schemas/2019-04-01/deploymentTemplate.json#",
    "contentVersion": "1.0.0.0",
    "parameters": {
        "workflows_scale_down_name": {
            "defaultValue": "${var.prefix}-${var.cluster_name}-workflow-scale-down",
            "type": "String"
        },
        "connections_keyvault_externalid": {
            "defaultValue": "/subscriptions/${var.subscription_id}/resourceGroups/${data.azurerm_resource_group.rg.name}/providers/Microsoft.Web/connections/${azurerm_key_vault.key_vault.name}-connection",
            "type": "String"
        }
    },
    "variables": {},
    "resources": [
        {
            "type": "Microsoft.Logic/workflows",
            "apiVersion": "2017-07-01",
            "name": "[parameters('workflows_scale_down_name')]",
            "location":  "[resourceGroup().location]",
            "identity": {
                "type": "SystemAssigned"
            },
            "properties": {
                "state": "Enabled",
                "definition": {
                    "$schema": "https://schema.management.azure.com/providers/Microsoft.Logic/schemas/2016-06-01/workflowdefinition.json#",
                    "contentVersion": "1.0.0.0",
                    "parameters": {
                        "$connections": {
                            "defaultValue": {},
                            "type": "Object"
                        }
                    },
                    "triggers": {
                        "scale-down-run-every-1-minute": {
                            "recurrence": {
                                "frequency": "Minute",
                                "interval": 1
                            },
                            "evaluatedRecurrence": {
                                "frequency": "Minute",
                                "interval": 1
                            },
                            "type": "Recurrence"
                        }
                    },
                    "actions": {}
                },
                "parameters": {
                    "$connections": {
                        "value": {
                            "keyvault": {
                                "connectionId": "[parameters('connections_keyvault_externalid')]",
                                "connectionName": "[concat('${azurerm_key_vault.key_vault.name}', '-connection')]",
                                "connectionProperties": {
                                    "authentication": {
                                        "type": "ManagedServiceIdentity"
                                    }
                                },
                                "id": "/subscriptions/${var.subscription_id}/providers/Microsoft.Web/locations/${data.azurerm_resource_group.rg.location}/managedApis/keyvault"
                            }
                        }
                    }
                }
            }
        }
    ],
   "outputs": {
      "LogicAppServiceIdentitylId": {
			"type": "string",
			"value": "[reference(concat('Microsoft.Logic/workflows/',parameters('workflows_scale_down_name')), '2017-07-01', 'Full').identity.principalId]"
		},
       "LogicAppId": {
        "type": "string",
        "value": "[resourceId('Microsoft.Logic/workflows', parameters('workflows_scale_down_name'))]"
      }
    }
}

TEMPLATE
  depends_on = [azurerm_resource_group_template_deployment.api_connections_template_deployment, azurerm_linux_virtual_machine_scale_set.vmss]
  lifecycle {
    ignore_changes = [template_content]
  }
}

locals {
  scale_down_logic_app_id          = jsondecode(azurerm_resource_group_template_deployment.workflow_scale_down_template_deployment.output_content).logicAppId.value
  scale_down_logic_app_identity_id = jsondecode(azurerm_resource_group_template_deployment.workflow_scale_down_template_deployment.output_content).logicAppServiceIdentitylId.value
  scale_up_logic_app_id            = jsondecode(azurerm_resource_group_template_deployment.workflow_scale_up_template_deployment.output_content).logicAppId.value
  scale_up_logic_app_identity_id   = jsondecode(azurerm_resource_group_template_deployment.workflow_scale_up_template_deployment.output_content).logicAppServiceIdentitylId.value
}

resource "azurerm_key_vault_access_policy" "logic-app-get-secret-permission" {
  key_vault_id = azurerm_key_vault.key_vault.id
  tenant_id    = data.azurerm_client_config.current.tenant_id
  object_id    = local.scale_down_logic_app_identity_id
  secret_permissions = [
    "Get",
  ]
  depends_on = [azurerm_key_vault.key_vault,azurerm_resource_group_template_deployment.workflow_scale_down_template_deployment]
}

resource "azurerm_role_assignment" "logic-app-key-vault-secrets-user" {
  scope                = azurerm_key_vault.key_vault.id
  role_definition_name = "Key Vault Secrets User"
  principal_id         = local.scale_down_logic_app_identity_id
  depends_on           = [azurerm_linux_function_app.function_app,azurerm_resource_group_template_deployment.workflow_scale_down_template_deployment]
}

resource "azurerm_role_assignment" "logic-app-storage-account-contributor" {
  scope                = azurerm_storage_account.deployment_sa.id
  role_definition_name = "Storage Blob Data Contributor"
  principal_id         = local.scale_down_logic_app_identity_id
  depends_on           = [azurerm_resource_group_template_deployment.workflow_scale_down_template_deployment]
}

resource "azurerm_logic_app_action_custom" "scale_down_logic_app_action_get_secret" {
  name         = "get-secret"
  logic_app_id = local.scale_down_logic_app_id
  body = <<BODY
{
  "runAfter": {},
  "type": "ApiConnection",
  "inputs": {
    "host": {
       "connection": {
          "name": "@parameters('$connections')['keyvault']['connectionId']"
       }
    },
    "method": "get",
    "path": "/secrets/@{encodeURIComponent('${azurerm_key_vault_secret.function_app_default_key.name}')}/value"
  },
   "runtimeConfiguration": {
      "secureData": {
          "properties": [
            "outputs"
      ]
    }
  }
}
BODY
  depends_on   = [azurerm_linux_function_app.function_app, azurerm_resource_group_template_deployment.workflow_scale_down_template_deployment]
}

resource "azurerm_logic_app_action_custom" "logic_app_action_fetch" {
  name         = "fetch"
  logic_app_id = local.scale_down_logic_app_id
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
  depends_on   = [azurerm_linux_function_app.function_app, azurerm_resource_group_template_deployment.workflow_scale_down_template_deployment,azurerm_logic_app_action_custom.scale_down_logic_app_action_get_secret]
}

resource "azurerm_logic_app_action_custom" "logic_app_action_scale_down" {
  name         = "scale-down"
  logic_app_id = local.scale_down_logic_app_id
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
  depends_on   = [azurerm_linux_function_app.function_app, azurerm_resource_group_template_deployment.workflow_scale_down_template_deployment,azurerm_logic_app_action_custom.logic_app_action_fetch]
}

resource "azurerm_logic_app_action_custom" "logic_app_action_terminate" {
  name         = "terminate"
  logic_app_id = local.scale_down_logic_app_id
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
  depends_on   = [azurerm_linux_function_app.function_app, azurerm_resource_group_template_deployment.workflow_scale_down_template_deployment,azurerm_logic_app_action_custom.logic_app_action_scale_down]
}

resource "azurerm_logic_app_action_custom" "logic_app_action_transient" {
  name         = "transient"
  logic_app_id = local.scale_down_logic_app_id
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
  depends_on = [azurerm_linux_function_app.function_app, azurerm_resource_group_template_deployment.workflow_scale_down_template_deployment,azurerm_logic_app_action_custom.logic_app_action_terminate]
}

resource "azurerm_monitor_diagnostic_setting" "logic_app_diagnostic_setting" {
  name                       = "${var.prefix}-${var.cluster_name}-workflow-diagnostic"
  target_resource_id         = local.scale_down_logic_app_id
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
  depends_on = [azurerm_resource_group_template_deployment.workflow_scale_down_template_deployment]
}

resource "azurerm_resource_group_template_deployment" "workflow_scale_up_template_deployment" {
  name                = "${var.prefix}-${var.cluster_name}-workflow-scale-up-deploy"
  resource_group_name = data.azurerm_resource_group.rg.name
  deployment_mode     = "Incremental"
  template_content    = <<TEMPLATE
  {
    "$schema": "https://schema.management.azure.com/schemas/2019-04-01/deploymentTemplate.json#",
    "contentVersion": "1.0.0.0",
    "parameters": {
        "workflows_scale_up_name": {
            "defaultValue": "${var.prefix}-${var.cluster_name}-workflow-scale-up",
            "type": "String"
        },
        "connections_keyvault_externalid": {
            "defaultValue": "/subscriptions/${var.subscription_id}/resourceGroups/${data.azurerm_resource_group.rg.name}/providers/Microsoft.Web/connections/${azurerm_key_vault.key_vault.name}-connection",
            "type": "String"
        }
    },
    "variables": {},
    "resources": [
        {
            "type": "Microsoft.Logic/workflows",
            "apiVersion": "2017-07-01",
            "name": "[parameters('workflows_scale_up_name')]",
            "location":  "[resourceGroup().location]",
            "identity": {
                "type": "SystemAssigned"
            },
            "properties": {
                "state": "Enabled",
                "definition": {
                    "$schema": "https://schema.management.azure.com/providers/Microsoft.Logic/schemas/2016-06-01/workflowdefinition.json#",
                    "contentVersion": "1.0.0.0",
                    "parameters": {
                        "$connections": {
                            "defaultValue": {},
                            "type": "Object"
                        }
                    },
                    "triggers": {
                        "scale-up-run-every-1-minute": {
                            "recurrence": {
                                "frequency": "Minute",
                                "interval": 1
                            },
                            "evaluatedRecurrence": {
                                "frequency": "Minute",
                                "interval": 1
                            },
                            "type": "Recurrence"
                        }
                    },
                    "actions": {}
                },
                "parameters": {
                    "$connections": {
                        "value": {
                            "keyvault": {
                                "connectionId": "[parameters('connections_keyvault_externalid')]",
                                "connectionName": "keyvault",
                                "connectionProperties": {
                                    "authentication": {
                                        "type": "ManagedServiceIdentity"
                                    }
                                },
                                "id": "/subscriptions/${var.subscription_id}/providers/Microsoft.Web/locations/${data.azurerm_resource_group.rg.location}/managedApis/keyvault"
                            }
                        }
                    }
                }
            }
        }
    ],
   "outputs": {
      "LogicAppServiceIdentitylId": {
			"type": "string",
			"value": "[reference(concat('Microsoft.Logic/workflows/',parameters('workflows_scale_up_name')), '2017-07-01', 'Full').identity.principalId]"
		},
       "LogicAppId": {
        "type": "string",
        "value": "[resourceId('Microsoft.Logic/workflows', parameters('workflows_scale_up_name'))]"
      }
    }
}

TEMPLATE
  depends_on = [azurerm_resource_group_template_deployment.api_connections_template_deployment, azurerm_linux_virtual_machine_scale_set.vmss]
  lifecycle {
    ignore_changes = [template_content]
  }
}

resource "azurerm_key_vault_access_policy" "scale-up-logic-app-get-secret-permission" {
  key_vault_id = azurerm_key_vault.key_vault.id
  tenant_id    = data.azurerm_client_config.current.tenant_id
  object_id    = local.scale_up_logic_app_identity_id
  secret_permissions = [
    "Get",
  ]
  depends_on = [azurerm_key_vault.key_vault,azurerm_resource_group_template_deployment.workflow_scale_up_template_deployment]
}

resource "azurerm_role_assignment" "scale-up-logic-app-key-vault-secrets-user" {
  scope                = azurerm_key_vault.key_vault.id
  role_definition_name = "Key Vault Secrets User"
  principal_id         = local.scale_up_logic_app_identity_id
  depends_on           = [azurerm_linux_function_app.function_app,azurerm_resource_group_template_deployment.workflow_scale_up_template_deployment]
}

resource "azurerm_role_assignment" "scale-up-logic-app-storage-account-contributor" {
  scope                = azurerm_storage_account.deployment_sa.id
  role_definition_name = "Storage Blob Data Contributor"
  principal_id         = local.scale_up_logic_app_identity_id
  depends_on           = [azurerm_resource_group_template_deployment.workflow_scale_up_template_deployment]
}

resource "azurerm_logic_app_action_custom" "scale_up_logic_app_action_get_secret" {
  name         = "get-secret"
  logic_app_id = local.scale_up_logic_app_id
  body = <<BODY
{
  "runAfter": {},
  "type": "ApiConnection",
  "inputs": {
    "host": {
       "connection": {
          "name": "@parameters('$connections')['keyvault']['connectionId']"
       }
    },
    "method": "get",
    "path": "/secrets/@{encodeURIComponent('${azurerm_key_vault_secret.function_app_default_key.name}')}/value"
  },
   "runtimeConfiguration": {
      "secureData": {
          "properties": [
            "outputs"
      ]
    }
  }
}
BODY
  depends_on   = [azurerm_linux_function_app.function_app, azurerm_resource_group_template_deployment.workflow_scale_up_template_deployment]
}

resource "azurerm_logic_app_action_custom" "logic_app_action_scale_up" {
  name         = "scale-up"
  logic_app_id = local.scale_up_logic_app_id
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
  depends_on   = [azurerm_linux_function_app.function_app, azurerm_resource_group_template_deployment.workflow_scale_up_template_deployment]
}

resource "azurerm_monitor_diagnostic_setting" "scale_up_logic_app_diagnostic_setting" {
  name                       = "${var.prefix}-${var.cluster_name}-workflow-scale-up-diagnostic"
  target_resource_id         = local.scale_up_logic_app_id
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
  depends_on = [azurerm_resource_group_template_deployment.workflow_scale_up_template_deployment]
}