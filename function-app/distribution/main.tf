locals {
  function_code_path     = "${abspath(path.module)}/../code/"
  function_app_code_hash = md5(join("", [for f in fileset(local.function_code_path, "**") : filemd5("${local.function_code_path}${f}")]))
  function_zip_path      = "/tmp/tf-function-app-${local.function_app_code_hash}.zip"
  function_triggers_path = "${abspath(path.module)}/../triggers"
}

resource "null_resource" "build_function_code" {
  triggers = {
    always_run = timestamp()
  }

  provisioner "local-exec" {
    command = <<EOT
    cd ${path.module}/../
    cd ${local.function_code_path}
    go mod tidy
    GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ${local.function_triggers_path}
    EOT
  }
}

data "archive_file" "function_zip" {
  type        = "zip"
  output_path = local.function_zip_path
  source_dir  = local.function_triggers_path
  depends_on  = [null_resource.build_function_code]
}

data "azurerm_resource_group" "rg" {
  name = var.resource_group_name
}

module "create-sa" {
  source   = "./create_sa"
  for_each = toset(var.regions["release"])

  rg_name = data.azurerm_resource_group.rg.name
  region  = each.key

  depends_on = [data.archive_file.function_zip]
}

module "upload-zip" {
  source   = "./upload_zip"
  for_each = toset(var.regions[var.dist])

  rg_name                = data.azurerm_resource_group.rg.name
  region                 = each.key
  function_app_zip_path  = local.function_zip_path
  function_app_code_hash = local.function_app_code_hash
  dist                   = var.dist

  depends_on = [module.create-sa]
}
