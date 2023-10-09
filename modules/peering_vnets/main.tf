data "azurerm_virtual_network" "vnet_id" {
  name                = var.vnet_name
  resource_group_name = var.vnet_rg_name
}

data "azurerm_virtual_network" "remote_vnet_ids" {
  count               = length(var.vnet_to_peering)
  name                = var.vnet_to_peering[count.index]["vnet"]
  resource_group_name = var.vnet_to_peering[count.index]["rg"]

}
resource "azurerm_virtual_network_peering" "peering" {
  count                        = length(var.vnet_to_peering)
  name                         = "${var.vnet_name}-peering-to-${var.vnet_to_peering[count.index]["vnet"]}"
  resource_group_name          = var.vnet_rg_name
  virtual_network_name         = var.vnet_name
  allow_virtual_network_access = true
  remote_virtual_network_id    = data.azurerm_virtual_network.remote_vnet_ids[count.index].id
}


resource "azurerm_virtual_network_peering" "peering2" {
  count                        = length(var.vnet_to_peering)
  name                         = "${var.vnet_to_peering[count.index]["vnet"]}-peering-to-${var.vnet_name}"
  resource_group_name          = data.azurerm_virtual_network.remote_vnet_ids[count.index].resource_group_name
  virtual_network_name         = data.azurerm_virtual_network.remote_vnet_ids[count.index].name
  remote_virtual_network_id    = data.azurerm_virtual_network.vnet_id.id
  allow_virtual_network_access = true
}
