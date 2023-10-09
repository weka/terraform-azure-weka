variable "service_principal_name" {
  description = "Name of the service principal"
  type        = string
  default     = "CIUser"
}

variable "description" {
  description = "Description of the service principal"
  type        = string
  default     = "Github CI user"
}

variable "azure_role_name" {
  description = "A unique UUID/GUID for this Role Assignment - one will be generated if not specified."
  type        = string
  default     = null
}

variable "azure_role_description" {
  description = "The description for this Role Assignment"
  type        = string
  default     = null
}

variable "assignments" {
  description = "The list of role assignments to this service principal"
  type = list(object({
    scope                = string
    role_definition_name = string
  }))
  default = [
    {
      scope                = "/subscriptions/d2f248b9-d054-477f-b7e8-413921532c2a"
      role_definition_name = "Owner"
    },
  ]
}
