variable "prefix" {
  type = string
  description = "Prefix for all resources"
  validation {
    condition     = can(regex("^[a-zA-Z][a-zA-Z\\-\\_0-9]{1,64}$", var.prefix))
    error_message = "Prefix name must start with letter, only contain letters, numbers, dashes, or underscores."
  }
}

variable "rg_name" {
  type = string
  description = "Name of existing resource group"
}

variable "address_space" {
  type = string
  description = "Address space that is used the virtual network."
}

variable "subnet_prefixes" {
  type = list(string)
  description = "List of address prefixes to use for the subnet"
}

variable "subnet_delegation" {
  type = string
  description = "Subnet delegation enables you to designate a specific subnet for an Azure PaaS service"
}

variable "cluster_name" {
  type = string
  description = "Cluster name"
  validation {
    condition     = can(regex("^[a-zA-Z][a-zA-Z\\-\\_0-9]{1,64}$", var.cluster_name))
    error_message = "Cluster name must start with letter, only contain letters, numbers, dashes, or underscores."
  }
}

variable "private_network" {
  type = bool
  description = "Should be true to enable private network, defaults to public networking"
}

variable "install_weka_url" {
  type = string
  description = "Url of weka tar installtion"
}

variable "apt_repo_url" {
  type = string
  description = "Url of private repo"
}

variable "vnet_to_peering" {
  type = list(object({
    vnet = string
    rg   = string
  }))
  description = "List of vent-name:resource-group-name to peer"
}

variable "install_ofed_url" {
  type = string
  description = "Link to blob of ofed version tgz"
}

variable "subscription_id" {
  type = string
  description = "Subscription id for deployment"
}

variable "instance_type" {
  type = string
  description = "The SKU which should be used for this virtual machine"
}

variable "set_obs_integration" {
  type = bool
  description = "Should be true to enable OBS integration with weka cluster"
}

variable "tiering_ssd_percent" {
  type = number
  description = "When OBS integration set to true , this parameter sets how much of the filesystem capacity should reside on SSD. For example, if this parameter is 20 and the total available SSD capacity is 20GB, the total capacity would be 100GB"
}

variable "cluster_size" {
  type = number
  description = "Weka cluster size"
}