variable "location" {
  type = string
  default = "East US"
}

variable "prefix" {
  type = string
  default = "ubuntu20"
}

variable "address_space" {
  type = list(string)
  default = ["11.0.0.0/16"]
}

variable "address_prefixes" {
  type = list(string)
  default = ["11.0.0.0/16"]
}

variable "vm_size" {
  type = string
  default = "Standard_D2s_v3"
}

variable "admin_username" {
  type = string
  default = "azureuser"
}

variable "ssh_public_key" {
  type = string
  default = "~/.ssh/weka_dev_ssh_key.pub"
}
