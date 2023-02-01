terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~>3.38.0"
    }
  }
  required_version = ">= 1.3.7"
}

provider "azurerm" {
  features {
  }
  subscription_id = "d2f248b9-d054-477f-b7e8-413921532c2a"
}