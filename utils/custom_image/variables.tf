variable "location" {
  type = string
  default = "East US"
}

variable "prefix" {
  type = string
  default = "weka-image"
}

variable "ofed_version" {
  type = string
  description = "The OFED driver version to for ubuntu 18."
  default = "5.7-1.0.2.0"
}

variable "subscription_id" {
  type = string
}

variable "rg_name" {
  type = string
}

variable "custom_vm_version" {
  type = string
  default = "1.0.0"
}

variable "tags" {
  type = map(string)
  default = {"creator": "tf"}
}

variable "custom_image_name" {
  type = string
  default = "weka-custom-image"
  description = "Name of created custom image"
}