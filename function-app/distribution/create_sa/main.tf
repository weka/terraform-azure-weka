resource "azurerm_storage_account" "sa" {
  name                     = "weka${var.region}"  // can only consist of lowercase letters and numbers, and must be between 3 and 24 characters long
  location                 = var.region
  resource_group_name      = var.rg_name
  account_kind             = "StorageV2"
  account_tier             = "Standard"
  account_replication_type = "ZRS"
  
  blob_properties {
    last_access_time_enabled = true
  }
}

resource "azurerm_storage_container" "container" {
  name                  = "weka-tf-functions-deployment-${var.region}"
  storage_account_name  = azurerm_storage_account.sa.name
  container_access_type = "blob"
  depends_on            = [azurerm_storage_account.sa]
}

resource "azurerm_storage_management_policy" "retention_policy" {
  storage_account_id = azurerm_storage_account.sa.id

  rule {
    name    = "weka-tf-functions-retention-rule-${var.region}"
    enabled = true
    filters {
      prefix_match = ["weka-tf-functions-deployment-${var.region}/dev"]
      blob_types   = ["blockBlob"]
    }
    actions {
      base_blob {
        tier_to_cool_after_days_since_last_access_time_greater_than    = 20
        delete_after_days_since_last_access_time_greater_than          = 30
        auto_tier_to_hot_from_cool_enabled = true
      }
    }
  }

  depends_on = [azurerm_storage_account.sa]
}
