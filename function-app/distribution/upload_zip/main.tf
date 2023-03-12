resource "azurerm_resource_group" "rg" {
  name     = "weka_tf_functions_${var.region}_rg"
  location = var.region
}

resource "azurerm_storage_account" "sa" {
  name                     = "wekatf${var.region}"  // can only consist of lowercase letters and numbers, and must be between 3 and 24 characters long
  location                 = azurerm_resource_group.rg.location
  resource_group_name      = azurerm_resource_group.rg.name
  account_kind             = "StorageV2"
  account_tier             = "Standard"
  account_replication_type = "ZRS"
}

resource "azurerm_storage_container" "container" {
  name                  = "weka-tf-functions-deployment-${var.region}"
  storage_account_name  = azurerm_storage_account.sa.name
  container_access_type = "blob"
  depends_on            = [azurerm_storage_account.sa]
}

data "azurerm_storage_account_sas" "sa_sas" {
  connection_string = azurerm_storage_account.sa.primary_connection_string
  https_only        = true

  resource_types {
    service   = true
    container = true
    object    = true
  }

  services {
    blob  = true
    queue = false
    table = false
    file  = false
  }

  start  = "${timestamp()}"
  expiry = timeadd("${timestamp()}", "20m")

  permissions {
    read    = true
    write   = true
    delete  = false
    list    = false
    add     = true
    create  = true
    update  = false
    process = false
    tag     = false
    filter  = false
  }
}

resource "null_resource" "function_app_code" {
  triggers = {
    zip_md5 = var.function_app_zip_md5
  }

  provisioner "local-exec" {
    command = <<EOT
DATE_NOW=$(date -Ru | sed 's/\+0000/GMT/')
AZ_VERSION="2021-12-02"
AZ_BLOB_TARGET="${azurerm_storage_account.sa.primary_blob_endpoint}${azurerm_storage_container.container.name}"
FILENAME="${var.function_app_zip_md5}.zip"
AZ_SAS_TOKEN="${data.azurerm_storage_account_sas.sa_sas.sas}"

curl --fail -X PUT -T ${var.function_app_zip_path} \
    -H "x-ms-date: $DATE_NOW" \
    -H "x-ms-blob-type: BlockBlob" \
    "$AZ_BLOB_TARGET/$FILENAME$AZ_SAS_TOKEN"
    EOT
  }

  depends_on             = [azurerm_storage_container.container]
}

