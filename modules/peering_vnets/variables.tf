variable "vnet_rg_name" {
  type        = string
  description = "Vnet resource group name"
}

variable "vnet_name" {
  type        = string
  description = "Vnet name"
}

variable "vnets_to_peer_to_deployment_vnet" {
  type = list(object({
    vnet = string
    rg   = string
  }))
  description = "List of vnet and rg for setting peering"
}
