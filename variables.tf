variable "prefix" {
  type = string
  description = "Prefix for all resources"
  validation {
    condition     = can(regex("^[a-zA-Z][a-zA-Z\\-\\_0-9]{1,64}$", var.prefix))
    error_message = "Prefix name must start with letter, only contain letters, numbers, dashes, or underscores."
  }
  default = "weka"
}

variable "rg_name" {
  type        = string
  description = "A predefined resource group in the Azure subscription."
}

variable "vnet_rg_name" {
  type        = string
  description = "Resource group name of vnet. Will be used when vnet_name is not provided."
  default     = ""
}

variable "subnet_prefix" {
  type        = string
  description = "Address prefixes to use for the subnet"
  default     = "10.0.2.0/24"
}

variable "allow_ssh_ranges" {
  type        = list(string)
  description = "Allow port 22, if not provided, i.e leaving the default empty list, the rule will not be included in the SG"
  default     = []
}

variable "allow_weka_api_ranges" {
  type        = list(string)
  description = "Allow port 14000, if not provided, i.e leaving the default empty list, the rule will not be included in the SG"
  default     = []
}

variable "address_space" {
  type        = string
  description = "The range of IP addresses the virtual network uses."
  default     = "10.0.0.0/16"
}

variable "vm_username" {
  type        = string
  description = "The user name for logging in to the virtual machines."
  default     = "weka"
}

variable "instance_type" {
  type        = string
  description = "The virtual machine type (sku) to deploy."
  default     = "Standard_L8s_v3"
}

variable "vnet_name" {
  type        = string
  description = "The virtual network name."
  default     = ""
}

variable "subnet_name" {
  type        = string
  description = "The subnet name."
  default     = ""
}

variable "cluster_size" {
  type        = number
  description = "The number of virtual machines to deploy."
  default     = 6

  validation {
    condition = var.cluster_size >= 6
    error_message = "Cluster size should be at least 6."
  }
}

variable "source_image_id" {
  type        = string
  default     = "/communityGalleries/WekaIO-d7d3f308-d5a1-4c45-8e8a-818aed57375a/images/ubuntu20.04/versions/latest"
  description = "Use weka custom image, ubuntu 20.04 with kernel 5.4 and ofed 5.8-1.1.2.1"
}

variable "sg_id" {
  type        = string
  description = "The security group id."
  default     = ""
}

variable "subnet_delegation" {
  type        = string
  description = "Subnet delegation enables you to designate a specific subnet for an Azure PaaS service."
  default     = "10.0.1.0/25"
}

variable "subnet_delegation_id" {
  type        = string
  description = "Subnet delegation id"
  default     = ""
}

variable "weka_version" {
  type        = string
  description = "The Weka version to deploy."
  default     = "4.2.1"
}

variable "get_weka_io_token" {
  type        = string
  description = "The token to download the Weka release from get.weka.io."
  default     = ""
  sensitive   = true
}

variable "cluster_name" {
  type = string
  description = "Cluster name"
  validation {
    condition     = can(regex("^[a-zA-Z][a-zA-Z\\-\\_0-9]{1,64}$", var.cluster_name))
    error_message = "Cluster name must start with letter, only contain letters, numbers, dashes, or underscores."
  }
  default = "poc"
}

variable "tags_map" {
  type = map(string)
  default = {"env": "dev", "creator": "tf"}
  description = "A map of tags to assign the same metadata to all resources in the environment. Format: key:value."
}

variable "ssh_public_key" {
  type        = string
  description = "Ssh public key to pass to vms."
  default     = null
}

variable "private_network" {
  type        = bool
  default     = false
  description = "Determines whether to enable a private or public network. The default is public network."
}

variable "assign_public_ip" {
  type        = bool
  default     = true
  description = "Determines whether to assign public ip."
}

variable "install_weka_url" {
  type        = string
  description = "The URL of the Weka release download tar file."
  default     = ""
}

variable "apt_repo_server" {
  type        = string
  description = "The URL of the apt private repository."
  default     = ""
}

variable "private_dns_zone_name" {
  type        = string
  description = "The private DNS zone name."
  default     = ""
}

variable "private_dns_rg_name" {
  type        = string
  description = "The private DNS zone resource group name. Required when private_dns_zone_name is set."
  default     = ""
}

variable "vnet_to_peering" {
  type = list(object({
    vnet = string
    rg   = string
  }))
  description = "List of vent-name:resource-group-name to peer"
  default     = []
}

variable "container_number_map" {
  type = map(object({
    compute  = number
    drive    = number
    frontend = number
    nvme     = number
    nics     = number
    memory   = list(string)
  }))
  description = "Maps the number of objects and memory size per machine type."
  default = {
    Standard_L8s_v3 = {
      compute  = 1
      drive    = 1
      frontend = 1
      nvme     = 1
      nics     = 4
      memory   = ["33GB","31GB"]
    },
    Standard_L16s_v3 = {
      compute  = 4
      drive    = 2
      frontend = 1
      nvme     = 2
      nics     = 8
      memory   = ["79GB","72GB"]
    },
    Standard_L32s_v3 = {
      compute  = 4
      drive    = 2
      frontend = 1
      nvme     = 4
      nics     = 8
      memory   = ["197GB","189GB"]
    },
    Standard_L48s_v3 = {
      compute  = 3
      drive    = 3
      frontend = 1
      nvme     = 6
      nics     = 8
      memory   = ["314GB","306GB"]
    },
    Standard_L64s_v3 = {
      compute  = 4
      drive    = 2
      frontend = 1
      nvme     = 8
      nics     = 8
      memory   = ["357GB","418GB"]
    }
  }
}

variable "default_disk_size" {
  type        = number
  default     = 48
  description = "The default disk size."
}

variable "traces_per_ionode" {
  default     = 10
  type        = number
  description = "The number of traces per ionode. Traces are low-level events generated by Weka processes and are used as troubleshooting information for support purposes."
}

variable "subscription_id" {
  type        = string
  description = "The subscription id for the deployment."
}

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

variable "function_app_log_level" {
  type = number
  default = 1
  description = "Log level for function app (from -1 to 5). See https://github.com/rs/zerolog#leveled-logging"

  validation {
    condition = var.function_app_log_level >= -1 && var.function_app_log_level <= 5
    error_message = "Allowed values for log level are from -1 to 5."
  }
}

variable "function_app_storage_account_prefix" {
  type = string
  description = "Weka storage account name prefix"
  default = "weka"
}

variable "function_app_storage_account_container_prefix" {
  type = string
  description = "Weka storage account container name prefix"
  default = "weka-tf-functions-deployment-"
}

variable "function_app_version" {
  type = string
  description = "Function app code version (hash)"
  default = "5a60e3dd08397aeadfc935f2a8a399e3"
}

variable "function_app_dist" {
  type = string
  description = "Function app code dist"
  default = "release"

  validation {
    condition = contains(["dev", "release"], var.function_app_dist)
    error_message = "Valid value is one of the following: dev, release."
  }
}

variable "install_cluster_dpdk" {
  type        = bool
  default     = true
  description = "Install weka cluster with DPDK"
}

variable "add_frontend_container" {
  type        = bool
  default     = true
  description = "Create cluster with FE containers"
}

################################################## obs variables ###################################################
variable "obs_name" {
  type = string
  default = ""
  description = "Name of existing obs storage account"
}

variable "obs_container_name" {
  type = string
  default = ""
  description = "Name of existing obs conatiner name"
}

variable "set_obs_integration" {
  type = bool
  default = false
  description = "Determines whether to enable object stores integration with the Weka cluster. Set true to enable the integration."
}

variable "blob_obs_access_key" {
  type = string
  description = "The access key of the existing Blob object store container."
  sensitive   = true
  default = ""
}

variable "tiering_ssd_percent" {
  type = number
  default = 20
  description = "When set_obs_integration is true, this variable sets the capacity percentage of the filesystem that resides on SSD. For example, for an SSD with a total capacity of 20GB, and the tiering_ssd_percent is set to 20, the total available capacity is 100GB."
}

############################### clients ############################
variable "clients_number" {
  type        = number
  description = "The number of client virtual machines to deploy."
  default     = 0
}

variable "client_instance_type" {
  type        = string
  description = "The client virtual machine type (sku) to deploy."
  default     = "Standard_D8_v5"
}

variable "client_nics_num" {
  type        = number
  description = "The client NICs number."
  default     = 2
}

variable "mount_clients_dpdk" {
  type        = bool
  default     = true
  description = "Mount weka clients in DPDK mode"
}

variable "placement_group_id" {
  type        = string
  default     = ""
  description = "Proximity placement group to use for the vmss. If not passed, will be created automatically."
}

variable "deployment_storage_account_name" {
  type        = string
  default     = ""
  description = "Name of exising deployment storage account"
}

variable "deployment_container_name" {
  type        = string
  default     = ""
  description = "Name of exising deployment container"
}

variable "deployment_storage_account_access_key" {
  type        = string
  description = "The access key of the existing Blob object store container."
  sensitive   = true
  default     = ""
}

variable "zone"{
  type        = string
  description = "The zone in which the resources should be created."
  default     = "1"
}

############################################### nfs protocol gateways variables ###################################################
variable "nfs_protocol_gateways_number" {
  type = number
  description = "The number of protocol gateway virtual machines to deploy."
  default     = 0
}

variable "nfs_protocol_gateway_secondary_ips_per_nic" {
  type        = number
  description = "Number of secondary IPs per single NIC per protocol gateway virtual machine."
  default     = 3
}

variable "nfs_protocol_gateway_instance_type" {
  type        = string
  description = "The protocol gateways' virtual machine type (sku) to deploy."
  default     = "Standard_D8_v5"
}

variable "nfs_protocol_gateway_nics_num" {
  type        = string
  description = "The protocol gateways' NICs number."
  default     = 2
}

variable "nfs_protocol_gateway_disk_size" {
  type        = number
  default     = 48
  description = "The protocol gateways' default disk size."
}

variable "nfs_protocol_gateway_frontend_cores_num" {
  type        = number
  default     = 1
  description = "The number of frontend cores on single protocol gateway machine."
}

variable "nfs_setup_protocol" {
  type        = bool
  description = "Config protocol, default if false"
  default     = false
}

############################################### smb protocol gateways variables ###################################################
variable "smb_protocol_gateways_number" {
  type = number
  description = "The number of protocol gateway virtual machines to deploy."
  default     = 0
}

variable "smb_protocol_gateway_secondary_ips_per_nic" {
  type        = number
  description = "Number of secondary IPs per single NIC per protocol gateway virtual machine."
  default     = 3
}

variable "smb_protocol_gateway_instance_type" {
  type        = string
  description = "The protocol gateways' virtual machine type (sku) to deploy."
  default     = "Standard_D8_v5"
}

variable "smb_protocol_gateway_nics_num" {
  type        = string
  description = "The protocol gateways' NICs number."
  default     = 2
}

variable "smb_protocol_gateway_disk_size" {
  type        = number
  default     = 48
  description = "The protocol gateways' default disk size."
}

variable "smb_protocol_gateway_frontend_cores_num" {
  type        = number
  default     = 1
  description = "The number of frontend cores on single protocol gateway machine."
}

variable "smb_setup_protocol" {
  type        = bool
  description = "Config protocol, default if false"
  default     = false
}

variable "smbw_enabled" {
  type        = bool
  default     = false
  description = "Enable SMBW protocol. This option should be provided before cluster is created to leave extra capacity for SMBW setup."
}

variable "smb_cluster_name" {
  type        = string
  description = "The name of the SMB setup."
  default     = "Weka-SMB"

  validation {
    condition     = length(var.smb_cluster_name) > 0
    error_message = "The SMB cluster name cannot be empty."
  }
}

variable "smb_domain_name" {
  type        = string
  description = "The domain to join the SMB cluster to."
  default     = ""
}

variable "smb_domain_netbios_name" {
  type        = string
  description = "The domain NetBIOS name of the SMB cluster."
  default     = ""
}

variable "smb_dns_ip_address" {
  type        = string
  description = "DNS IP address"
  default     = ""
}

variable "smb_share_name" {
  type       = string
  description = "The name of the SMB share"
  default     = "default"
}

variable "proxy_url" {
  type        = string
  description = "Weka home proxy url"
  default     = ""
}

variable "weka_home_url" {
  type        = string
  description = "Weka Home url"
  default     = ""
}
