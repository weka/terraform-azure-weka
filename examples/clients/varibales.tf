variable "weka_version" {
  type = string
  description = "The Weka version to deploy."
  default = "4.1.0.71"
}

variable "get_weka_io_token" {
  type = string
  description = "The token to download the Weka release from get.weka.io."
  default = ""
  sensitive = true
}

variable "nics" {
  type = number
  default = 2
  description = "Number of nics to set on each client vm"
}

variable "sg_name" {
  type = string
  description = "Security group name for clients"
}

variable "linux_vm_image" {
  type        = map(string)
  description = "The default azure vm image reference."
  default     = {
    publisher = "Canonical"
    offer     = "UbuntuServer"
    sku       = "18.04-LTS"
    version   = "latest"
  }
}

variable "custom_image_id" {
  type        = string
  description = "Custom image id"
  default     = null
}

variable "prefix" {
  type = string
  description = "The prefix for all the resource names. For example, the prefix for your system name."
  default = "weka"
}

variable "rg_name" {
  type = string
  description = "A predefined resource group in the Azure subscription."
}

variable "vm_username" {
  type = string
  description = "The user name for logging in to the virtual machines."
  default = "weka"
}

variable "instance_type" {
  type = string
  description = "The virtual machine type (sku) to deploy."
  default = "Standard_L8s_v3"
}

variable "vnet_name" {
  type = string
  description = "The virtual network name."
}

variable "clients_name" {
  type = string
  description = "The clients name."
}

variable "subnet_name" {
  type = string
  description = "The subnet names list."
}

variable "clients_size" {
  type = number
  description = "The number of virtual machines to deploy."
  default = 2
}

variable "ssh_public_key" {
  type = string
  description = "The path to the VM public key. If it is not set, the key is auto-generated. If it is set, also set the ssh_private_key."
  default = null
}

variable "ssh_private_key" {
  type = string
  description = "The path to the VM private key. If it is not set, the key is auto-generated. If it is set, also set the ssh_private_key. The private key used for connecting to the deployed virtual machines to initiate the clusterization of Weka."
  default = null
}

variable "ofed_version" {
  type = string
  description = "The OFED driver version to for ubuntu 18."
  default = "5.9-0.5.6.0"
}

variable "subscription_id" {
  type = string
  description = "The subscription id for the deployment."
}

variable "install_dpdk" {
  type        = bool
  default     = true
  description = "Install weka cluster with DPDK"
}

variable "install_ofed" {
  type        = bool
  default     = true
  description = "Install ofed for weka cluster with dpdk configuration"
}

variable "lb_url" {
  type = string
}