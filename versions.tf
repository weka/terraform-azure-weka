terraform {
  required_version = ">= 1.4.6"
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~>4.6.0"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "~>4.0.4"
    }
    local = {
      source  = "hashicorp/local"
      version = "~>2.4.0"
    }
  }
}
