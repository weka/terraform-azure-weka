variable "subscription_id" {
  type        = string
  description = "Subscription id for deployment"
}


variable "regions" {
  type        = list(string)
  description = "List of supported regions"

  default = [
    "eastus",
    # "westus",
    # "westeurope",
  ]
}
