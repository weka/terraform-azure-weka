resource "azurerm_resource_group" "rg" {
  count    = var.rg_name == null ? 1: 0
  location = var.location
  name     = "${var.prefix}-rg"
}

data azurerm_resource_group "get-rg" {
  count = var.rg_name != null ? 1 : 0
  name  = var.rg_name
}

locals {
  rg       = var.rg_name == null ? azurerm_resource_group.rg[0].name : data.azurerm_resource_group.get-rg[0].name
  location = var.rg_name == null ? azurerm_resource_group.rg[0].location : data.azurerm_resource_group.get-rg[0].location
}

resource "azurerm_shared_image_gallery" "shared_image_gallery" {
  name                = replace("wekaGallery","/\\W|_|\\s/","")
  resource_group_name = local.rg
  location            = local.location
  description         = "Shared weka custom images"
  tags                = var.tags
}

data "azurerm_images" "image" {
  resource_group_name = local.rg
}

resource "azurerm_shared_image" "shared_image" {
  name                = "${var.prefix}-shared-definition"
  gallery_name        = azurerm_shared_image_gallery.shared_image_gallery.name
  resource_group_name = local.rg
  location            = local.location
  os_type             = "Linux"
  tags                = var.tags
  specialized         = false
  hyper_v_generation = "V2"
  identifier {
    publisher = "CanonicalWeka"
    offer     = "UbuntuServerWeka"
    sku       = "18.04-lts-gen2"
  }
}

resource "azurerm_shared_image_version" "shared_image_version" {
  name                = var.custom_vm_version
  gallery_name        = azurerm_shared_image_gallery.shared_image_gallery.name
  image_name          = azurerm_shared_image.shared_image.name
  resource_group_name = local.rg
  location            = local.location
  managed_image_id    = "/subscriptions/${var.subscription_id}/resourceGroups/${local.rg}/providers/Microsoft.Compute/images/${var.custom_image_name}"
  exclude_from_latest = false
  tags                = var.tags
  target_region {
    name                   = azurerm_shared_image.shared_image.location
    regional_replica_count = 100
    storage_account_type   = "Standard_LRS"
  }
  depends_on = [azurerm_shared_image.shared_image]
}