locals {
  function_code_path     = "${abspath(path.module)}/../code/"
  function_app_code_hash = md5(join("", [for f in fileset(local.function_code_path, "**") : filemd5("${local.function_code_path}${f}")]))
  function_zip_path      = "/tmp/tf-function-app-${local.function_app_code_hash}.zip"
  function_app_tf_path   = "${abspath(path.module)}/../tf-function-app/"
}

resource "null_resource" "build_function_code" {
  triggers = {
    dir_md5 = local.function_app_code_hash
  }

  provisioner "local-exec" {
    command = <<EOT
    cd ${path.module}/../
    cd ${local.function_code_path}
    go mod tidy
    GOOS=linux GOARCH=amd64 go build -o ${local.function_app_tf_path}
    EOT
  }
}

resource "null_resource" "make_function_zip" {
  triggers = {
    dir_md5 = local.function_app_code_hash
  }
  provisioner "local-exec" {
    command = (
      !fileexists(local.function_zip_path)
      ? "zip -r ${local.function_zip_path} ${local.function_app_tf_path}"
      : "echo \"File ${local.function_zip_path} already exists.\""
    )
  }
  depends_on = [null_resource.build_function_code]
}

module "upload-zip" {
  source   = "./upload_zip"
  for_each = toset(var.regions)

  region                 = each.key
  function_app_zip_path  = local.function_zip_path
  function_app_code_hash = local.function_app_code_hash

  depends_on = [null_resource.make_function_zip]
}
