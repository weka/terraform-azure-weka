terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~>3.44.0"
    }
  }
  required_version = ">= 1.3.7"
}

provider "azurerm" {
  features {
  }
  subscription_id = var.subscription_id
}