output "function_app_zip_md5" {
  value       = local.function_app_code_hash
  description = "Function app code dir MD5"
}

output "function_app_zip_filepath" {
  value       = local.function_zip_path
  description = "Function app code zip path"
}
