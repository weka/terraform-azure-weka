variable "backend_lb_ip" {
  type        = string
  description = "The backend load balancer ip address."
}

variable "nics" {
  type        = number
  default     = 2
  description = "Number of nics to set on each client vm"
}

variable "source_image_id" {
  type        = string
  description = "Use weka custom image, ubuntu 20.04 with kernel 5.4 and ofed 5.8-1.1.2.1"
}

variable "rg_name" {
  type        = string
  description = "A predefined resource group in the Azure subscription."
}

variable "vm_username" {
  type        = string
  description = "The user name for logging in to the virtual machines."
  default     = "weka"
}

variable "instance_type" {
  type        = string
  description = "The virtual machine type (sku) to deploy."
}

variable "vnet_name" {
  type        = string
  description = "The virtual network name."
}

variable "clients_name" {
  type        = string
  description = "The clients name."
}

variable "subnet_name" {
  type        = string
  description = "The subnet names."
}

variable "clients_number" {
  type        = number
  description = "The number of virtual machines to deploy."
  default     = 2
}

variable "ssh_public_key" {
  type        = string
  description = "The VM public key. If it is not set, the keys are auto-generated."
}

variable "apt_repo_url" {
  type        = string
  default     = ""
  description = "The URL of the apt private repository."
}

variable "mount_clients_dpdk" {
  type        = bool
  default     = true
  description = "Install weka cluster with DPDK"
}

variable "nics_map" {
  type = map(number)
  default = {
    Standard_L8s_v3  = 4
    Standard_L16s_v3 = 8
  }
}

variable "ppg_id" {
  type        = string
  description = "Placement proximity group id."
}

variable "assign_public_ip" {
  type        = bool
  default     = true
  description = "Determines whether to assign public ip."
}

variable "vnet_rg_name" {
  type        = string
  description = "Resource group name of vnet"
}

variable "sg_id" {
  type        = string
  description = "Security group id"
}
