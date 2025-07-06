variable "backend_lb_ip" {
  type        = string
  description = "The backend load balancer ip address."
  default     = ""
}

variable "frontend_container_cores_num" {
  type        = number
  default     = 1
  description = "Number of nics to set on each client vm"
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
  description = "Ssh public key to pass to vms."
}

variable "apt_repo_server" {
  type        = string
  default     = ""
  description = "The URL of the apt private repository."
}

variable "clients_use_dpdk" {
  type        = bool
  default     = true
  description = "Mount weka clients in DPDK mode."
}

variable "ppg_id" {
  type        = string
  default     = null
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

variable "tags_map" {
  type        = map(string)
  default     = {}
  description = "A map of tags to assign the same metadata to all resources in the environment. Format: key:value."
}

variable "custom_data" {
  type        = string
  description = "Custom data to pass to the instances. Deprecated, use `custom_data_post_mount` instead."
  default     = ""
}

variable "custom_data_pre_mount" {
  type        = string
  description = "Custom data to pass to the instances, will run before weka agent install and mount."
  default     = ""
}

variable "custom_data_post_mount" {
  type        = string
  description = "Custom data to pass to the instances, will run after weka agent install and mount."
  default     = ""
}


variable "vmss_name" {
  type        = string
  description = "The name of the backends virtual machine scale set."
}

variable "vm_identity_name" {
  type        = string
  description = "The name of the user assigned identity for the client VMs."
  default     = ""
}

variable "arch" {
  type    = string
  default = null
}

variable "source_image_id" {
  type        = string
  description = "Use weka custom image, ubuntu 20.04 with kernel 5.4 and ofed 5.8-1.1.2.1"
  default     = ""
}

variable "use_vmss" {
  type        = bool
  description = "Use VMSS"
  default     = false
}

variable "root_volume_size" {
  type        = number
  description = "The client's root volume size in GB"
  default     = null
}
