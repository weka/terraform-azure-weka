terraform {
  required_version = ">= 1.4.6"
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~>3.114.0"
    }
  }
  # https://learn.microsoft.com/en-us/azure/developer/terraform/store-state-in-azure-storage?tabs=azure-cli
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
