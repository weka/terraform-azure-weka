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

data "azurerm_storage_account_blob_container_sas" "sa_sas" {
  connection_string = azurerm_storage_account.sa.primary_connection_string
  container_name    = azurerm_storage_container.container.name
  https_only        = true

  start  = "${timestamp()}"
  expiry = timeadd("${timestamp()}", "20m")

  permissions {
    read   = true
    add    = true
    create = true
    write  = true
    delete = false
    list   = true
  }
}

resource "null_resource" "function_app_code" {
  triggers = {
    upload_when_changed = "${var.dist}/${var.function_app_code_hash}"
  }

  provisioner "local-exec" {
    command = <<EOT
    DATE_NOW=$(date -Ru | sed 's/\+0000/GMT/')
    AZ_VERSION="2021-12-02"
    AZ_BLOB_TARGET="${azurerm_storage_account.sa.primary_blob_endpoint}${azurerm_storage_container.container.name}"
    FILENAME="${var.dist}/${var.function_app_code_hash}.zip"
    AZ_SAS_TOKEN="${data.azurerm_storage_account_blob_container_sas.sa_sas.sas}"

    curl --fail -X PUT -T ${var.function_app_zip_path} \
        -H "x-ms-date: $DATE_NOW" \
        -H "x-ms-blob-type: BlockBlob" \
        "$AZ_BLOB_TARGET/$FILENAME$AZ_SAS_TOKEN"
    EOT
  }

  depends_on = [azurerm_storage_container.container]
}

