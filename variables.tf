variable "prefix" {
  type        = string
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

variable "create_nat_gateway" {
  type        = bool
  default     = false
  description = "NAT needs to be created when no public ip is assigned to the backend, to allow internet access"
}

variable "allow_ssh_cidrs" {
  type        = list(string)
  description = "Allow port 22, if not provided, i.e leaving the default empty list, the rule will not be included in the SG"
  default     = []
}

variable "allow_weka_api_cidrs" {
  type        = list(string)
  description = "Allow connection to port 14000 on weka backends from specified CIDRs, by default no CIDRs are allowed. All ports (including 14000) are allowed within Vnet"
  default     = []
}

variable "sg_custom_ingress_rules" {
  type = list(object({
    from_port  = string
    to_port    = string
    protocol   = string
    cidr_block = string
  }))
  default     = []
  description = "Custom inbound rules to be added to the security group."
}

variable "address_space" {
  type        = string
  description = "The range of IP addresses the virtual network uses."
  default     = "10.0.0.0/16"
}

variable "vm_username" {
  type        = string
  description = "Provided as part of output for automated use of terraform, in case of custom AMI and automated use of outputs replace this with user that should be used for ssh connection"
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
    condition     = var.cluster_size >= 6
    error_message = "Cluster size should be at least 6."
  }
}

variable "source_image_id" {
  type        = string
  default     = "/communityGalleries/WekaIO-ddbef83d-dec1-42d0-998a-3c083f1450b7/images/weka_custom_image/versions/1.0.1"
  description = "Use weka custom image, ubuntu 20.04 with kernel 5.4 and ofed 5.8-1.1.2.1"
}

variable "sg_id" {
  type        = string
  description = "The security group id."
  default     = ""
}

variable "function_app_subnet_delegation_cidr" {
  type        = string
  description = "Subnet delegation enables you to designate a specific subnet for an Azure PaaS service."
  default     = "10.0.1.0/25"
}

variable "function_app_subnet_delegation_id" {
  type        = string
  description = "Required to specify if subnet_name were used to specify pre-defined subnets for weka. Function subnet delegation requires an additional subnet, and in the case of pre-defined networking this one also should be pre-defined"
  default     = ""
}

variable "logic_app_subnet_delegation_id" {
  type        = string
  default     = ""
  description = "Required to specify if subnet_name were used to specify pre-defined subnets for weka. Logicapp subnet delegation requires an additional subnet, and in the case of pre-defined networking this one also should be pre-defined"
}

variable "logic_app_subnet_delegation_cidr" {
  type        = string
  default     = "10.0.3.0/25"
  description = "Subnet delegation enables you to designate a specific subnet for an Azure PaaS service."
}

variable "logic_app_identity_name" {
  type        = string
  description = "The user assigned identity name for the logic app (if empty - new one is created)."
  default     = ""
}

variable "weka_version" {
  type        = string
  description = "The Weka version to deploy."
  default     = ""
}

variable "get_weka_io_token" {
  type        = string
  description = "The token to download the Weka release from get.weka.io."
  default     = ""
  sensitive   = true
}

variable "cluster_name" {
  type        = string
  description = "Cluster name"
  validation {
    condition     = can(regex("^[a-zA-Z][a-zA-Z\\-\\_0-9]{1,64}$", var.cluster_name))
    error_message = "Cluster name must start with letter, only contain letters, numbers, dashes, or underscores."
  }
  default = "poc"
}

variable "tags_map" {
  type        = map(string)
  default     = {}
  description = "A map of tags to assign the same metadata to all resources in the environment. Format: key:value."
}

variable "ssh_public_key" {
  type        = string
  description = "Ssh public key to pass to vms."
  default     = null
}

variable "assign_public_ip" {
  type        = string
  default     = "auto"
  description = "Determines whether to assign public IP to all instances deployed by TF module. Includes backends, clients and protocol gateways."
  validation {
    condition     = var.assign_public_ip == "true" || var.assign_public_ip == "false" || var.assign_public_ip == "auto"
    error_message = "Allowed assign_public_ip values: [\"true\", \"false\", \"auto\"]."
  }
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

variable "private_dns_zone_use" {
  type        = bool
  description = "Determines whether to use private DNS zone. Required for LB record creation."
  default     = true
}

variable "vnets_to_peer_to_deployment_vnet" {
  type = list(object({
    vnet = string
    rg   = string
  }))
  description = "List of vent-name:resource-group-name to peer"
  default     = []
}

variable "containers_config_map" {
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
      memory   = ["33GB", "31GB"]
    },
    Standard_L16s_v3 = {
      compute  = 4
      drive    = 2
      frontend = 1
      nvme     = 2
      nics     = 8
      memory   = ["79GB", "72GB"]
    },
    Standard_L32s_v3 = {
      compute  = 4
      drive    = 2
      frontend = 1
      nvme     = 4
      nics     = 8
      memory   = ["197GB", "189GB"]
    },
    Standard_L48s_v3 = {
      compute  = 3
      drive    = 3
      frontend = 1
      nvme     = 6
      nics     = 8
      memory   = ["314GB", "306GB"]
    },
    Standard_L64s_v3 = {
      compute  = 4
      drive    = 2
      frontend = 1
      nvme     = 8
      nics     = 8
      memory   = ["357GB", "384GB"]
    },
    Standard_L80s_v3 = {
      compute  = 4
      drive    = 2
      frontend = 1
      nvme     = 8
      nics     = 8
      memory   = ["384GB", "384GB"]
    },
    Standard_L8as_v3 = {
      compute  = 1
      drive    = 1
      frontend = 1
      nvme     = 1
      nics     = 4
      memory   = ["29GB", "29GB"]
    },
    Standard_L16as_v3 = {
      compute  = 4
      drive    = 2
      frontend = 1
      nvme     = 2
      nics     = 8
      memory   = ["72GB", "73GB"]
    },
    Standard_L32as_v3 = {
      compute  = 4
      drive    = 2
      frontend = 1
      nvme     = 4
      nics     = 8
      memory   = ["190GB", "190GB"]
    },
    Standard_L48as_v3 = {
      compute  = 3
      drive    = 3
      frontend = 1
      nvme     = 6
      nics     = 8
      memory   = ["308GB", "308GB"]
    },
    Standard_L64as_v3 = {
      compute  = 4
      drive    = 2
      frontend = 1
      nvme     = 8
      nics     = 8
      memory   = ["384GB", "384GB"]
    },
    Standard_L80as_v3 = {
      compute  = 4
      drive    = 2
      frontend = 1
      nvme     = 8
      nics     = 8
      memory   = ["384GB", "384GB"]
    }
  }
  validation {
    condition     = alltrue([for m in flatten([for i in values(var.containers_config_map) : (flatten(i.memory))]) : tonumber(trimsuffix(m, "GB")) <= 384])
    error_message = "Compute memory can not be more then 384GB"
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
  type        = number
  default     = 2
  description = "Cluster data protection level."
  validation {
    condition     = var.protection_level == 2 || var.protection_level == 4
    error_message = "Allowed protection_level values: [2, 4]."
  }
}

variable "stripe_width" {
  type        = number
  default     = -1
  description = "Stripe width = cluster_size - protection_level - 1 (by default)."
  validation {
    condition     = var.stripe_width == -1 || var.stripe_width >= 3 && var.stripe_width <= 16
    error_message = "The stripe_width value can take values from 3 to 16."
  }
}

variable "hotspare" {
  type        = number
  default     = 1
  description = "Number of hotspares to set on weka cluster. Refer to https://docs.weka.io/weka-system-overview/ssd-capacity-management#hot-spare"
}

variable "function_app_log_level" {
  type        = number
  default     = 1
  description = "Log level for function app (from -1 to 5). See https://github.com/rs/zerolog#leveled-logging"

  validation {
    condition     = var.function_app_log_level >= -1 && var.function_app_log_level <= 5
    error_message = "Allowed values for log level are from -1 to 5."
  }
}

variable "function_app_storage_account_prefix" {
  type        = string
  description = "Weka storage account name prefix"
  default     = "weka"
}

variable "function_app_storage_account_container_prefix" {
  type        = string
  description = "Weka storage account container name prefix"
  default     = "weka-tf-functions-deployment-"
}

variable "function_app_version" {
  type        = string
  description = "Function app code version (hash)"
  default     = "c1e7430d91c6592c6b95e467ce9929fd"
}

variable "function_app_dist" {
  type        = string
  description = "Function app code dist"
  default     = "dev"

  validation {
    condition     = contains(["dev", "release"], var.function_app_dist)
    error_message = "Valid value is one of the following: dev, release."
  }
}

variable "function_app_identity_name" {
  type        = string
  description = "The user assigned identity name for the function app (if empty - new one is created)."
  default     = ""
}

variable "vmss_identity_name" {
  type        = string
  description = "The user assigned identity name for the vmss instances (if empty - new one is created)."
  default     = ""
}

variable "install_cluster_dpdk" {
  type        = bool
  default     = true
  description = "Install weka cluster with DPDK"
}

variable "set_dedicated_fe_container" {
  type        = bool
  default     = false
  description = "Create cluster with FE containers"
}

variable "log_analytics_workspace_id" {
  type        = string
  description = "The Log Analytics workspace id."
  default     = ""
}

variable "application_insights_name" {
  type        = string
  description = "The Application Insights name."
  default     = ""
}

variable "application_insights_rg_name" {
  type        = string
  description = "The Application Insights resource group name."
  default     = ""
}

variable "enable_application_insights" {
  type        = bool
  default     = true
  description = "Enable Application Insights."
}

variable "application_insights_instrumentation_key" {
  type        = string
  description = "The Application Insights instrumentation key."
  default     = ""
}

variable "create_lb" {
  type        = bool
  default     = true
  description = "Create backend and UI load balancers for weka cluster."
}

################################################## obs variables ###################################################
variable "tiering_obs_name" {
  type        = string
  default     = ""
  description = "Name of existing obs storage account"
}

variable "tiering_enable_obs_integration" {
  type        = bool
  default     = false
  description = "Determines whether to enable object stores integration with the Weka cluster. Set true to enable the integration."
}

variable "tiering_enable_ssd_percent" {
  type        = number
  default     = 20
  description = "When set_obs_integration is true, this variable sets the capacity percentage of the filesystem that resides on SSD. For example, for an SSD with a total capacity of 20GB, and the tiering_ssd_percent is set to 20, the total available capacity is 100GB."
}

variable "tiering_obs_container_name" {
  type        = string
  default     = ""
  description = "Name of existing obs container name."
}

variable "tiering_blob_obs_access_key" {
  type        = string
  description = "The access key of the existing Blob object store container. If not provided, new obs will be created with given name (tiering_obs_name)."
  sensitive   = true
  default     = ""
}

variable "tiering_obs_target_ssd_retention" {
  type        = number
  description = "Target retention period (in seconds) before tiering to OBS (how long data will stay in SSD). Default is 86400 seconds (24 hours)."
  default     = 86400
}

variable "tiering_obs_start_demote" {
  type        = number
  description = "Target tiering cue (in seconds) before starting upload data to OBS (turning it into read cache). Default is 10 seconds."
  default     = 10
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
  default     = ""
}

variable "client_source_image_id" {
  type        = string
  description = "Use weka custom image, ubuntu 20.04 with kernel 5.4 and ofed 5.8-1.1.2.1 / ubuntu arm 20.04 with kernel 5.4 and ofed 5.9-0.5.6.0"
  default     = ""
}

variable "client_arch" {
  type        = string
  default     = null
  description = "Use arch for ami id, value can be arm64/x86_64."
  validation {
    condition     = var.client_arch == "arm64" || var.client_arch == "x86_64" || var.client_arch == null
    error_message = "Allowed client_arch values: [\"arm64\", \"x86_64\", null]."
  }
}

variable "clients_use_dpdk" {
  type        = bool
  default     = true
  description = "Mount weka clients in DPDK mode"
}

variable "client_identity_name" {
  type        = string
  description = "The user assigned identity name for the client instances (if empty - new one is created)."
  default     = ""
}

variable "client_placement_group_id" {
  type        = string
  description = "The client instances placement group id. Backend placement group can be reused. If not specified placement group will be created automatically"
  default     = ""
}

variable "client_frontend_cores" {
  type        = number
  description = "The client NICs number."
  default     = 1
}

variable "clients_custom_data" {
  type        = string
  description = "Custom data to pass to the client instances"
  default     = ""
}

variable "clients_use_vmss" {
  type        = bool
  default     = false
  description = "Use VMSS for clients"
}

variable "placement_group_id" {
  type        = string
  default     = ""
  description = "Proximity placement group to use for the vmss. If not passed, will be created automatically."
}

variable "vmss_single_placement_group" {
  type        = bool
  default     = true
  description = "Sets single_placement_group option for vmss. If true, a scale set is composed of a single placement group, and has a range of 0-100 VMs."
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

variable "deployment_file_share_name" {
  type        = string
  default     = ""
  description = "Name of exising deployment file share. Will use '<deployment_storage_account_name>-share' name if not provided."
}

variable "deployment_function_app_code_blob" {
  type        = string
  description = "The path to the function app code blob file."
  default     = ""
}

variable "zone" {
  type        = string
  description = "The zone in which the resources should be created."
  default     = null
}

variable "protocol_gateways_identity_name" {
  type        = string
  description = "The user assigned identity name for the protocol gateways instances (if empty - new one is created)."
  default     = ""
}

variable "nfs_deployment_container_name" {
  type        = string
  default     = ""
  description = "Name of exising protocol deployment container"
}

############################################### nfs protocol gateways variables ###################################################
variable "nfs_protocol_gateways_number" {
  type        = number
  description = "The number of protocol gateway virtual machines to deploy."
  default     = 0
}

variable "nfs_protocol_gateway_secondary_ips_per_nic" {
  type        = number
  description = "Number of secondary IPs per single NIC per protocol gateway virtual machine."
  default     = 0

  validation {
    condition     = var.nfs_protocol_gateway_secondary_ips_per_nic == 0
    error_message = "Secondary (floating) IPs are currently not supported for Azure NFS protocol gateways."
  }
}

variable "nfs_protocol_gateway_instance_type" {
  type        = string
  description = "The protocol gateways' virtual machine type (sku) to deploy."
  default     = "Standard_D8_v5"
}

variable "nfs_protocol_gateway_disk_size" {
  type        = number
  default     = 48
  description = "The protocol gateways' default disk size."
}

variable "nfs_protocol_gateway_fe_cores_num" {
  type        = number
  default     = 1
  description = "The number of frontend cores on single protocol gateway machine."
}

variable "nfs_setup_protocol" {
  type        = bool
  description = "Config protocol, default if false"
  default     = false
}

variable "nfs_interface_group_name" {
  type        = string
  description = "Interface group name."
  default     = "weka-ig"

  validation {
    condition     = length(var.nfs_interface_group_name) <= 11
    error_message = "The interface group name should be up to 11 characters long."
  }
}

############################################### smb protocol gateways variables ###################################################
variable "smb_protocol_gateways_number" {
  type        = number
  description = "The number of protocol gateway virtual machines to deploy."
  default     = 0
}

variable "smb_protocol_gateway_secondary_ips_per_nic" {
  type        = number
  description = "Number of secondary IPs per single NIC per protocol gateway virtual machine."
  default     = 0
}

variable "smb_protocol_gateway_instance_type" {
  type        = string
  description = "The protocol gateways' virtual machine type (sku) to deploy."
  default     = "Standard_D8_v5"
}

variable "smb_protocol_gateway_disk_size" {
  type        = number
  default     = 48
  description = "The protocol gateways' default disk size."
}

variable "smb_protocol_gateway_fe_cores_num" {
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
  default     = true
  description = "Enable SMBW protocol. This option should be provided before cluster is created to leave extra capacity for SMBW setup."
}

variable "smb_cluster_name" {
  type        = string
  description = "The name of the SMB setup."
  default     = "Weka-SMB"

  validation {
    condition     = length(var.smb_cluster_name) > 0 && length(var.smb_cluster_name) <= 15
    error_message = "The SMB cluster name must be between 1 and 15 characters long."
  }
}

variable "smb_domain_name" {
  type        = string
  description = "The domain to join the SMB cluster to."
  default     = ""
}

variable "smb_dns_ip_address" {
  type        = string
  description = "DNS IP address"
  default     = ""
}

variable "smb_create_private_dns_resolver" {
  type        = bool
  default     = false
  description = "Create dns resolver for smb with outbound rule"
}

variable "smb_dns_resolver_subnet_delegation_cidr" {
  type        = string
  default     = "10.0.4.0/28"
  description = "Cidr of dns resolver of subnet, for SMB"
}

variable "smb_dns_resolver_subnet_delegation_id" {
  type        = string
  default     = ""
  description = "Required to specify if subnet_id were used to specify pre-defined for SMB dns resolver subnet, requires an additional subnet, '/subscriptions/../resourceGroups/../providers/Microsoft.Network/virtualNetworks/../subnets/..'"
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

############################################### S3 protocol gateways variables ###################################################
variable "s3_protocol_gateways_number" {
  type        = number
  description = "The number of protocol gateway virtual machines to deploy."
  default     = 0
}

variable "s3_protocol_gateway_instance_type" {
  type        = string
  description = "The protocol gateways' virtual machine type (sku) to deploy."
  default     = "Standard_D8_v5"
}

variable "s3_protocol_gateway_disk_size" {
  type        = number
  default     = 48
  description = "The protocol gateways' default disk size."
}

variable "s3_protocol_gateway_fe_cores_num" {
  type        = number
  default     = 1
  description = "The number of frontend cores on single protocol gateway machine."
}

variable "s3_setup_protocol" {
  type        = bool
  description = "Config protocol, default if false"
  default     = false
}

#### private blob
variable "weka_tar_storage_account_id" {
  type    = string
  default = ""
}

variable "function_access_restriction_enabled" {
  type        = bool
  default     = false
  description = "Allow public access, Access restrictions apply to inbound access to internal vent"
}

variable "script_post_cluster_creation" {
  type        = string
  description = "Script to run after cluster creation"
  default     = ""
}

variable "script_pre_start_io" {
  type        = string
  description = "Script to run before starting IO"
  default     = ""
}

variable "clusterization_target" {
  type        = number
  description = "The clusterization target"
  default     = null
}

variable "user_data" {
  type        = string
  description = "User data to pass to vms."
  default     = ""
}

variable "debug_down_backends_removal_timeout" {
  type        = string
  default     = "3h"
  description = "Don't change this value without consulting weka support team. Timeout for removing down backends. Valid time units are ns, us (or µs), ms, s, m, h."
}

variable "storage_account_public_network_access" {
  type        = string
  description = "Public network access to the storage accounts."
  default     = "Enabled"

  validation {
    condition     = contains(["Enabled", "Disabled", "EnabledForVnet"], var.storage_account_public_network_access)
    error_message = "Allowed values: [\"Enabled\", \"Disabled\", \"EnabledForVnet\"]."
  }
}

variable "storage_account_allowed_ips" {
  type        = list(string)
  description = "IP ranges to allow access from the internet or your on-premises networks to storage accounts."
  default     = []
}

variable "create_storage_account_private_links" {
  type        = bool
  default     = false
  description = "Create private links for storage accounts (needed in case if public network access for the storage account is disabled)."
}

variable "storage_blob_private_dns_zone_name" {
  type        = string
  description = "The private DNS zone name for the storage account (blob)."
  default     = "privatelink.blob.core.windows.net"
}

variable "read_function_zip_from_storage_account" {
  type        = bool
  default     = false
  description = "Read function app zip from storage account (is read from public distribution storage account by default)."
}

variable "key_vault_purge_protection_enabled" {
  type        = bool
  default     = false
  description = "Enable purge protection for the key vault."
}

variable "set_default_fs" {
  type        = bool
  description = "Set the default filesystem which will use the full available capacity"
  default     = true
}

variable "post_cluster_setup_script" {
  type        = string
  description = "A script to run after the cluster is up"
  default     = ""
}
