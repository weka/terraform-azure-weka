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

variable "vnet_name" {
  type        = string
  description = "The name of the virtual network."
}

variable "cidr_range" {
  type        = string
  description = "The address space that is used by the subnet."
}

variable "delegation_name" {
  type        = string
  description = "The name of the subnet delegation."
  default     = "subnet-delegation"
}

variable "service_endpoints" {
  type        = list(string)
  description = "The list of service endpoints."
  default     = ["Microsoft.Storage", "Microsoft.KeyVault", "Microsoft.Web"]
}
