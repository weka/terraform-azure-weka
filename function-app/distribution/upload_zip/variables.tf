variable "rg_name" {
  type = string
  description = "Resource group name."
}

variable "region" {
  type = string
  description = "Azure location (region)"
}

variable "function_app_zip_path" {
  type = string
  description = "An absolute path to the code zip on the local system."
}

variable "function_app_code_hash" {
  type = string
  description = "The MD5 checksum of function app dir."
}

variable "dist" {
  type = string
  description = "Distribution option ('dev' or 'release')"
  default = "dev"
  
  validation {
    condition = contains(["dev", "release"], var.dist)
    error_message = "Valid value is one of the following: dev, release."
  }
}
