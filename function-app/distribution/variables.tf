variable "resource_group_name" {
  type        = string
  description = "The name of Azure resource group"
  default     = "weka-tf-functions"
}

variable "subscription_id" {
  type        = string
  description = "Subscription id for deployment"
}

variable "supported_regions_file_path" {
  type        = string
  description = "Path to supported regions file"
  default     = "supported_regions/release.txt"
}
