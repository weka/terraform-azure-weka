output "custom-vm-id" {
  value = azurerm_shared_image_version.shared_image_version.id
}

output "gallery-name" {
  value = azurerm_shared_image_gallery.shared_image_gallery.name
}

output "image-name" {
  value = azurerm_shared_image.shared_image.name
}

output "managed_image_id" {
  value = azurerm_shared_image_version.shared_image_version.managed_image_id
}

output "image-identifier" {
  value = azurerm_shared_image.shared_image.identifier[0]
}