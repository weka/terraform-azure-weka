variable "get_weka_io_token" {
  type        = string
  sensitive   = true
  description = "Get get.weka.io token for downloading weka"
}

variable "subscription_id" {
  type        = string
  description = "Subscription id for deployment"
}
