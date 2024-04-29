data "azurerm_user_assigned_identity" "this" {
  count               = var.vm_identity_name != "" ? 1 : 0
  name                = var.vm_identity_name
  resource_group_name = data.azurerm_resource_group.rg.name
}

resource "azurerm_user_assigned_identity" "this" {
  count               = var.vm_identity_name == "" ? 1 : 0
  location            = data.azurerm_resource_group.rg.location
  name                = "${var.clients_name}-identity"
  resource_group_name = data.azurerm_resource_group.rg.name
}

resource "azurerm_role_definition" "nics_reader" {
  count       = var.vm_identity_name == "" ? 1 : 0
  name        = "${var.clients_name}-vmss-nics-reader"
  scope       = data.azurerm_resource_group.rg.id
  description = "Can read backends VMSS network interfaces"

  permissions {
    actions = [
      "Microsoft.Compute/virtualMachineScaleSets/networkInterfaces/read",
    ]
    not_actions = []
  }

  assignable_scopes = [data.azurerm_resource_group.rg.id]
}

resource "azurerm_role_assignment" "reader" {
  count                = var.vm_identity_name == "" ? 1 : 0
  scope                = data.azurerm_resource_group.rg.id
  role_definition_name = azurerm_role_definition.nics_reader[0].name
  principal_id         = azurerm_user_assigned_identity.this[0].principal_id
}
