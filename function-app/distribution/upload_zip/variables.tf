variable "region" {
  type = string
  description = "Azure location (region)"
}

variable "function_app_zip_path" {
  type = string
  description = "An absolute path to the code zip on the local system."
}

variable "function_app_zip_md5" {
  type = string
  description = "The MD5 checksum of output archive file."
}
