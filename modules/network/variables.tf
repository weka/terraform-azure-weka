variable "prefix" {
  type = string
  description = "The prefix for all the resource names. For example, the prefix for your system name."
  default = "weka"
}

variable "rg_name" {
  type = string
  description =  "A predefined resource group in the Azure subscription."
}

variable "address_space" {
  type = string
  description = "The range of IP addresses the virtual network uses."
  default = "10.0.0.0/16"
}

variable "subnet_prefix" {
  type        = string
  description = "Address prefixes to use for the subnet."
  default     = "10.0.2.0/24"
}

variable "tags_map" {
  type = map(string)
  default = {"env": "dev", "creator": "tf"}
  description = "A map of tags to assign the same metadata to all resources in the environment. Format: key:value."
}

variable "vnet_name" {
  type        = string
  default     = ""
  description = "The VNet name, if exists."
}

variable "subnet_name" {
  type        = string
  default     = ""
  description = "Subnet name, if exist."
}

variable "private_network" {
  type        = bool
  default     = false
  description = "Determines whether to enable a private or public network. The default is public network."
}

variable "allow_ssh_ranges" {
  type        = list(string)
  description = "A list of IP addresses that can use ssh connection with a public network deployment."
  default     = []
}

variable "vnet_rg_name" {
  type        = string
  default     = ""
  description = "Resource group name of vnet"
}

variable "private_dns_rg_name" {
  type        = string
  description = "The private DNS zone resource group name. Required when private_dns_zone_name is set."
  default     = ""
}

variable "private_dns_zone_name" {
  type        = string
  description = "The private DNS zone name."
  default     = ""
}