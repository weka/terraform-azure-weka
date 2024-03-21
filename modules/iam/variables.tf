variable "rg_name" {
  type        = string
  description = "A predefined resource group in the Azure subscription."
}

variable "prefix" {
  type        = string
  description = "Prefix for all resources"
  validation {
    condition     = can(regex("^[a-zA-Z][a-zA-Z\\-\\_0-9]{1,64}$", var.prefix))
    error_message = "Prefix name must start with letter, only contain letters, numbers, dashes, or underscores."
  }
}

variable "cluster_name" {
  type        = string
  description = "Cluster name"
  validation {
    condition     = can(regex("^[a-zA-Z][a-zA-Z\\-\\_0-9]{1,64}$", var.cluster_name))
    error_message = "Cluster name must start with letter, only contain letters, numbers, dashes, or underscores."
  }
}

variable "vmss_identity_name" {
  type        = string
  description = "The user assigned identity name for the vmss instances (if empty - new one is created)."
  default     = ""
}

variable "function_app_identity_name" {
  type        = string
  description = "The user assigned identity name for the function app (if empty - new one is created)."
  default     = ""
}

variable "logic_app_identity_name" {
  type        = string
  description = "The user assigned identity name for the logic app (if empty - new one is created)."
  default     = ""
}

variable "logic_app_storage_account_id" {
  type        = string
  description = "The id of the storage account for the logic app."
}

variable "key_vault_id" {
  type        = string
  description = "The id of the Azure Key Vault."
}

variable "weka_tar_storage_account_id" {
  type    = string
  default = ""
}

variable "deployment_storage_account_id" {
  type        = string
  description = "The id of the storage account for the deployment."
}

variable "deployment_container_name" {
  type        = string
  description = "The name of the container for the deployment."
}

variable "tiering_enable_obs_integration" {
  type        = bool
  description = "Enable OBS integration for tiering."
}

variable "tiering_obs_name" {
  type        = string
  default     = ""
  description = "Name of existing obs storage account."
}

variable "obs_container_name" {
  type        = string
  description = "The name of the container for the OBS."
}

variable "nfs_protocol_gateways_number" {
  type        = number
  description = "The number of NFS protocol gateways."
}

variable "nfs_deployment_container_name" {
  type        = string
  description = "The name of the container for the NFS deployment."
}
