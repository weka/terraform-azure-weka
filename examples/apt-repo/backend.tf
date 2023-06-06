terraform {
  backend "azurerm" {
    resource_group_name  = "weka-tf"
    storage_account_name = "wekatfbackendsa"
    container_name       = "tfstate"
    key                  = "dev-repo20.terraform.tfstate"
    subscription_id      = "d2f248b9-d054-477f-b7e8-413921532c2a"
  }
}