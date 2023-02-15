
variable "protection_level" {
  type = number
  default = 2
  description = "Cluster data protection level."
  validation {
    condition     = var.protection_level == 2 || var.protection_level == 4
    error_message = "Allowed protection_level values: [2, 4]."
  }
}

variable "stripe_width" {
  type = number
  default = -1
  description = "Stripe width = cluster_size - protection_level - 1 (by default)."
  validation {
    condition     = var.stripe_width == -1 || var.stripe_width >= 3 && var.stripe_width <= 16
    error_message = "The stripe_width value can take values from 3 to 16."
  }
}

variable "hotspare" {
  type = number
  default = 1
  description = "Hot-spare value."
}