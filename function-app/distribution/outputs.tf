output "function_app_zip_md5" {
  value       = data.archive_file.function_zip.output_md5
  description = "Function app code zip MD5"
}
