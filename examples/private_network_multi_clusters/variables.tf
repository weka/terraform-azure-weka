variable "prefix" {
  type = string
  description = "Prefix for all resources"
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

variable "private_network" {
  type = bool
  description = "Should be true to enable private network, defaults to public networking"
}

variable "install_weka_url" {
  type = string
  description = "Url of weka tar"
}

variable "apt_repo_url" {
  type = string
  description = "Url of private repo"
}

variable "clusters_list" {
  type = list(string)
  description = "list of clusters name"
  validation {
    condition     = can(regex("^[a-zA-Z][a-zA-Z\\-\\_0-9]{1,64}$", var.clusters_list))
    error_message = "Cluster name must start with letter, only contain letters, numbers, dashes, or underscores."
  }
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