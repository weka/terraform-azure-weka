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
  default = ""
}

variable "subnet_prefixes" {
  type = list(string)
  description = "A list of address prefixes to use for the subnet."
  default = []
}

variable "subnet_delegation" {
  type = string
  description = "Subnet delegation enables you to designate a specific subnet for an Azure PaaS service"
}

variable "tags_map" {
  type = map(string)
  default = {"env": "dev", "creator": "tf"}
  description = "A map of tags to assign the same metadata to all resources in the environment. Format: key:value."
}

variable "vnet_name" {
  type = string
  default = null
  description = "The VNet name, if exists."
}

variable "subnets_name_list" {
  type = list(string)
  default = []
  description = "A list of subnet names, if exist."
}

variable "private_network" {
  type = bool
  default = false
  description = "Determines whether to enable a private or public network. The default is public network."
}

variable "sg_public_ssh_ips" {
  type        = list(string)
  description = "A list of IP addresses that can use ssh connection with a public network deployment."
  default = ["0.0.0.0/0"]
}