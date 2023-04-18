data "azurerm_storage_account" "sa" {
  name                = "weka${var.region}"
  resource_group_name = var.rg_name
}

data "azurerm_storage_container" "container" {
  name                 = "weka-tf-functions-deployment-${var.region}"
  storage_account_name = data.azurerm_storage_account.sa.name
}

data "azurerm_storage_account_blob_container_sas" "sa_sas" {
  connection_string = data.azurerm_storage_account.sa.primary_connection_string
  container_name    = data.azurerm_storage_container.container.name
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
    AZ_BLOB_TARGET="${data.azurerm_storage_account.sa.primary_blob_endpoint}${data.azurerm_storage_container.container.name}"
    FILENAME="${var.dist}/${var.function_app_code_hash}.zip"
    AZ_SAS_TOKEN="${data.azurerm_storage_account_blob_container_sas.sa_sas.sas}"

    curl --fail -X PUT -T ${var.function_app_zip_path} \
        -H "x-ms-date: $DATE_NOW" \
        -H "x-ms-blob-type: BlockBlob" \
        "$AZ_BLOB_TARGET/$FILENAME$AZ_SAS_TOKEN"
    EOT
  }

  depends_on = [data.azurerm_storage_container.container]
}

