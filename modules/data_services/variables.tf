variable "rg_name" {
  type        = string
  description = "A predefined resource group in the Azure subscription."
}

variable "location" {
  type        = string
  description = "The Azure region to deploy all resources to."
}

variable "instance_type" {
  type        = string
  description = "The virtual machine type (sku) to deploy."
}

variable "vnet_rg_name" {
  type        = string
  description = "Resource group name of vnet"
}

variable "vnet_name" {
  type        = string
  description = "The virtual network name."
}

variable "subnet_name" {
  type        = string
  description = "The subnet names."
}

variable "tags_map" {
  type        = map(string)
  default     = {}
  description = "A map of tags to assign the same metadata to all resources in the environment. Format: key:value."
}

variable "data_services_number" {
  type        = number
  description = "The number of virtual machines to deploy as data services."
}

variable "data_services_name" {
  type        = string
  description = "The data services group name."
}

variable "cluster_name" {
  type        = string
  description = "The cluster name."
}

variable "vm_username" {
  type        = string
  description = "The user name for logging in to the virtual machines."
  default     = "weka"
}

variable "ssh_public_key" {
  type        = string
  description = "The VM public key. If it is not set, the keys are auto-generated."
}

variable "assign_public_ip" {
  type        = bool
  default     = true
  description = "Determines whether to assign public ip."
}

variable "sg_id" {
  type        = string
  description = "Security group id."
}

variable "source_image_id" {
  type        = string
  description = "Use weka custom image, ubuntu 20.04 with kernel 6.8.0. Ignored if image_sku is set."
}

variable "image_sku" {
  type        = string
  default     = null
  description = "Azure Marketplace image SKU (e.g., '22_04-lts-gen2' for Ubuntu 22.04 LTS). Required when using marketplace images. Must be paired with image_offer. Use underscores, not dots. When set, source_image_id is ignored."
}

variable "image_offer" {
  type        = string
  default     = null
  description = "Azure Marketplace image offer (e.g., '0001-com-ubuntu-server-jammy' for Ubuntu 22.04 LTS). Required when image_sku is set. Common offers: 0001-com-ubuntu-server-focal (20.04), 0001-com-ubuntu-server-jammy (22.04)."
}

variable "image_version" {
  type        = string
  default     = "latest"
  description = "Azure Marketplace image version (e.g., 'latest' or a specific version). Only used when image_sku is set. Use 'latest' for the most recent version, or pin to a specific version for reproducibility."
}

variable "apt_repo_server" {
  type        = string
  default     = ""
  description = "The URL of the apt private repository."
}

variable "disk_size" {
  type        = number
  description = "The data services' weka disk size."
}

variable "root_volume_size" {
  type        = number
  default     = null
  description = "The data services' root disk size."
}

variable "key_vault_id" {
  type        = string
  description = "The id of the Azure Key Vault."
}

variable "weka_tar_storage_account_id" {
  type    = string
  default = ""
}

variable "vm_identity_name" {
  type        = string
  description = "The name of the user assigned identity for the data services VMs."
  default     = ""
}

variable "deploy_function_url" {
  type        = string
  description = "The URL of deploy function from function app."
}

variable "report_function_url" {
  type        = string
  description = "The URL of report function from function app."
}

variable "function_app_default_key" {
  type        = string
  description = "The default key of the function app."
}
