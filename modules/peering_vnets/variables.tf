variable "prefix" {
  type = string
  description = "Prefix for all resources"
  default = "weka"
}

variable "vnet_rg_name" {
  type = string
  description = "Vnet resource group name"
}

variable "vnet_name" {
  type = string
  description = "Vnet name"
}

variable "vnet_to_peering" {
  type = list(object({
      vnet = string
      rg   = string
  }))
  description = "List of vnet and rg for setting peering"
}
