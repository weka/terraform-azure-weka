variable "prefix" {
  type        = string
  description = "Prefix for all resources"
}

variable "cluster_name" {
  type        = string
  description = "Cluster name"
}

variable "rg_name" {
  type        = string
  description = "A predefined resource group in the Azure subscription."
}

variable "subnet_id" {
  type        = string
  description = "The ID of the cluster subnet."
}

variable "location" {
  type        = string
  description = "The Azure region to deploy all resources to."
}

variable "logic_app_subnet_delegation_id" {
  type        = string
  description = "Required to specify if subnet_name were used to specify pre-defined subnets for weka. Logicapp subnet delegation requires an additional subnet, and in the case of pre-defined networking this one also should be pre-defined"
}

variable "storage_account_name" {
  type        = string
  description = "The name of the storage account."
}

variable "logic_app_identity_id" {
  type        = string
  description = "The ID of the managed identity for the logic app"
}

variable "logic_app_identity_principal" {
  type        = string
  description = "The principal ID of the managed identity for the logic app"

}

variable "restricted_inbound_access" {
  type        = bool
  description = "Restrict inbound access to internal VNet"
}

variable "function_app_name" {
  type        = string
  description = "The name of the function app."
}

variable "function_app_id" {
  type        = string
  description = "The ID of the function app."
}

variable "function_app_key" {
  type        = string
  description = "The key of the function app."
}

variable "key_vault_id" {
  type        = string
  description = "The id of the Azure Key Vault."
}

variable "key_vault_uri" {
  type        = string
  description = "The URI of the Azure Key Vault."
}

variable "use_secured_storage_account" {
  type        = bool
  description = "Use secured storage account with logic app."
  default     = false
}
