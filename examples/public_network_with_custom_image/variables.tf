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
  description = "address space that is used the virtual network."
}

variable "subnet_prefixes" {
  type = list(string)
  description = "List of address prefixes to use for the subnet"
}

variable "subnet_delegation" {
  type = string
  description = "Subnet delegation enables you to designate a specific subnet for an Azure PaaS service"
}

variable "get_weka_io_token" {
  type = string
  sensitive = true
  description = "Get get.weka.io token for downloading weka"
}

variable "cluster_name" {
  type = string
  description = "Cluster name"
  validation {
    condition     = can(regex("^[a-zA-Z][a-zA-Z\\-\\_0-9]{1,64}$", var.cluster_name))
    error_message = "Cluster name must start with letter, only contain letters, numbers, dashes, or underscores."
  }
}

variable "subscription_id" {
  type = string
  description = "Subscription id for deployment"
}

variable "instance_type" {
  type = string
  description = "The SKU which should be used for this virtual machine"
}

variable "custom_image_id" {
  type = string
  description = "Custom image id; /subscriptions/<subscription id>/resourceGroups/<rg name>/providers/Microsoft.Compute/images/<image name>"
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

variable "single_placement_group" {
  type = bool
  description = "Limited Single Placement Group, which means the number of instances will be capped at 100 Virtual Machines"
}