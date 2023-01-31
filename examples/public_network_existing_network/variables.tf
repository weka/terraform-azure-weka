variable "prefix" {
  type = string
  description = "Prefix for all resources"
}

variable "rg_name" {
  type = string
  description = "Name of existing resource group"
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

variable "vnet_name" {
  type = string
  description = "Name of existing vnet"
}

variable "subnets_name_list" {
  type = list(string)
  default = []
  description = "Names of existing subnets list"
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