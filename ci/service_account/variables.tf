variable "service_principal_name" {
  description = "Name of the service principal"
  default     = "CIUser"
}

variable "description" {
  description = "Description of the service principal"
  default     = "Github CI user"
}

variable "role_definition_name" {
  description = "built-in role for the service principal"
  default     = null
}

variable "azure_role_name" {
  description = "A unique UUID/GUID for this Role Assignment - one will be generated if not specified."
  default     = null
}

variable "azure_role_description" {
  description = "The description for this Role Assignment"
  default     = null
}

variable "subscription_id" {
  description = "Subscription ID from Azure"
  default     = null
}


variable "assignments" {
  description = "The list of role assignments to this service principal"
  type        = list(object({
    scope                = string
    role_definition_name = string
  }))
  default = [
    {
      scope                = "/subscriptions/subscription_id"
      role_definition_name = "Owner"
    },
  ]
}
