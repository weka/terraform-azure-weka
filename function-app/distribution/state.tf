terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~>3.43.0"
    }
  }
  backend "azurerm" {
    resource_group_name  = "weka-tf-functions"
    storage_account_name = "wekatfstate"
    container_name       = "weka-tf-functions-state-container"
    key                  = "terraform.tfstate"
  }

}

provider "azurerm" {
  subscription_id = var.subscription_id
  features {}
}
