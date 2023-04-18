variable "resource_group_name" {
  type        = string
  description = "The name of Azure resource group"
  default     = "weka-tf-functions"
}

variable "subscription_id" {
  type        = string
  description = "Subscription id for deployment"
}

variable "dist" {
  type        = string
  description = "Distribution option ('dev' or 'release')"
  default     = "dev"

  validation {
    condition     = contains(["dev", "release"], var.dist)
    error_message = "Valid value is one of the following: dev, release."
  }
}

variable "regions" {
  type        = map(list(string))
  description = "Map of lists of supported regions"

  default = {
    "dev" = ["eastus", "uksouth"],
    "release" = [
      "brazilsouth",
      "canadacentral",
      "centralus",
      "eastus",
      "eastus2",
      //"northcentralus", // does not include the Zone-redundant storage (ZRS) blob
      "southcentralus",
      //"westus", // Sku: Standard_ZRS, Kind: StorageV2 is not available in zone
      "westus2",
      "westus3",
      "francecentral",
      "germanywestcentral",
      "northeurope",
      "westeurope",
      "uksouth",
      "swedencentral",
      "qatarcentral",
      "australiaeast",
      //"australiasoutheast", // does not include the Zone-redundant storage (ZRS) blob
      "centralindia",
      "japaneast",
      "eastasia",
      "southeastasia",
    ]
  }
}
