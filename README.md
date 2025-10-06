# Azure-weka deployment Terraform package
The Weka cluster on Azure provides a fast and scalable platform to run, for example, performance-intensive applications and hybrid cloud workflows. It can also be used for object stores, tiering, and snapshots using the Azure Blob service.
The provided Azure-Weka Terraform package contains modules and examples you can customize according to your deployment needs. The installation is based on applying the customized Terraform variables file to a predefined Azure subscription.
Applying the Terraform variables file performs the following:
- Creates resources in a predefined resource group, such as virtual machines, network interfaces, function app, load balancer, and more.
- Deploys Azure virtual machines.
- Installs the Weka software.
- Configures the Weka cluster.

<br> You can find [here](https://github.com/weka/terraform-azure-weka-essential) our essential deployment which creates only vms and placement group.

## Weke deployment prerequisites:
- resource group for deployment
- vnet
- subnet
- 2 subnets delegations - one for our function app and one for our logic app
- security group (needs to allow network inside the vnet)
- dns zone

## Resource group
We have 3 variables that define resource group:
- rg_name
- vnet_rg_name
- private_dns_rg_name
#### rg_name:
The resource group were weka cluster and all necessary resources will be deployed.
#### vnet_rg_name:
The resource group of the vnet and subnet.
#### private_dns_rg_name:
The private DNS zone resource group name.

<br>If `vnet_rg_name` isn't set by the user, we assume that the
vnet and subnet resource group is the as the weka deployment resource group.
<br> i.e we assume `vnet_rg_name = rg_name`
<br>Same goes for `private_dns_rg_name`.
<br>If `private_dns_rg_name` isn't set by the user, we assume that the
private dns resource group name is the same as the weka deployment resource group.
<br> i.e we assume `private_dns_rg_name = rg_name`

## Network deployment options
This weka deployment can use existing network, or create network resources (vmet, subnet, security group etc.) automatically.
<br>Check our [examples](examples).
<br>In case you want to use an existing vnet and subnet, you **must** provide them.
<br>**Example**:
```hcl
vnet_name           = "my-vnet"
subnet_name         = "my-subnet"
```
<br>In case you want to use an existing subnet delegations, you **must** provide them.
<br>**Example**:
```hcl
function_app_subnet_delegation_id      = "subnet-delegation-id1"
logic_app_subnet_delegation_id         = "subnet-delegation-id2"
```
<br>In case you want to use an existing security group, you **must** provide it.
<br>**Example**:
```hcl
sg_id      = "sg-id"
```
<br>In case you want to use a dns zone, you **must** provide it.
<br>**Example**:
```hcl
private_dns_zone_name             = "myDns.private.net"
private_dns_rg_name               = "myResourceGroup"
```
**If you don't pass these params, we will automatically create the network resources for you.**

## Storage account
### We create/use the following storage accounts
- Logic app storage account - Stores the logic app configuration. Created by our module.
- Deployment storage account - Stores the deployment states (cluster and also NFS if configured). Created by our module if not provided.
- Weka OBS storage account - Created by our function app if OBS is configured and OBS storage account is not provided.

### Storage account networking options
```hcl
variable "storage_account_public_network_access" {
  type        = string
  description = "Public network access to the storage accounts."
  default     = "Enabled"

  validation {
    condition     = contains(["Enabled", "Disabled", "EnabledForVnet"], var.storage_account_public_network_access)
    error_message = "Allowed values: [\"Enabled\", \"Disabled\", \"EnabledForVnet\"]."
  }
}
```
- `Enabled`: By default, the storage account is created with public network access enabled.
- `EnabledForVnet`: The storage account is created with public network access enabled, but only for the specified virtual network.
  - Access should be enabled for the vnet, function app subnet delegation.
  - File share is required (can provide existing via `deployment_file_share_name` or it will be auto-created in case if `storage_account_allowed_ips` are provided).
  - `storage_account_allowed_ips`: required to allow creating the logic app storage account with the required config and function app file share.
  - if `storage_account_allowed_ips` if not provided, scale down and autoscaling will not be supported and the file share needs to be created by the user.
  - OBS storage account: if created by our module only the regular [OBS](#OBS) config is required. If provided by the user needs to have the Vnet enabled.
- `Disabled`: The storage account is created with public network access disabled.
  - Scale down and autoscaling is not supported.
  - Pre created deployment storage account is required.
  - File share is required (`deployment_file_share_name`).
  - Blob and file endpoints and private links are required. It can be created by our module if `create_storage_account_private_links` is provided or by the user. In case if there are existing private endpoints and `create_storage_account_private_links` is not set, `storage_blob_private_dns_zone_name` can be also set to specify private DNS zone for blob resource (uses Azure-recommended name as default value).
  - OBS storage account: if created by our module only the regular [OBS](#OBS) config is required. If provided by the user, blob and file endpoints and private links are required.
They can be created by our module if `create_storage_account_private_links` is provided.

## Usage example
```hcl
provider "azurerm" {
  subscription_id = "mySubscriptionId"
  partner_id      = "f13589d1-f10d-4c3b-ae42-3b1a8337eaf1"
  features {
  }
}

terraform {
  backend "azurerm" {
    resource_group_name  = "myStateResourceGroup"
    storage_account_name = "myStateStorageAccount"
    container_name       = "myStateContainer"
    key                  = "state.terraform.tfstate"
  }
}


module "deploy_weka" {
  source                            = "weka/weka/azure"
  version                           = "3.0.5"
  prefix                            = "weka"
  rg_name                           = "myResourceGroup"
  vnet_name                         = "weka-vpc-0"
  vnet_rg_name                      = "myVnetResourceGroup"
  subnet_name                       = "weka-subnet-0"
  sg_id                             = "security-group-id"
  get_weka_io_token                 = "get_weka_io_token"
  cluster_name                      = "myCluster"
  function_app_subnet_delegation_id = "subnet-delegation-id1"
  logic_app_subnet_delegation_id    = "subnet-delegation-id2"
  set_obs_integration               = true
  instance_type                     = "Standard_L8s_v3"
  cluster_size                      = 6
  assign_public_ip                  = false
  subscription_id                   = "mySubscriptionId"
  private_dns_zone_name             = "myDns.private.net"
  private_dns_rg_name               = "myResourceGroup"
}

output "deploy_weka_output" {
  value = module.deploy_weka
}
```

### Private network deployment:
#### To enable public ip assignment:
```hcl
assign_public_ip   = true
```
#### Vms with no internet outbound:
In case your vms don't have internet access, you should supply weka tar file url, apt repo url and service endpoints:
```hcl
apt_repo_url = "..."
install_weka_url = "..."
```
#### Service endpoints:
The deployment and delegation subnets must include the following service endpoints:
- "Microsoft.Storage"
- "Microsoft.KeyVault"
- "Microsoft.Web"

The delegation subnets must include the following action action:
```hcl
service_delegation {
  name    = "Microsoft.Web/serverFarms"
  actions = ["Microsoft.Network/virtualNetworks/subnets/action"]
}
```

## Weka custom image
As you can see via `source_image_id` variable, we use our own custom image.
This is a community image that we created and uploaded to azure.
In case you would like to view how we created the image you can find it [here](https://github.com/weka/terraform-azure-weka-custom-image).
You can as well create it on your own subscription and use it.


## Ssh keys
The username for ssh into vms is `weka`.
<br />

We allow passing an existing public key:
```hcl
ssh_public_key = "..."
```
If public key isn't passed we will create it for you and store the private key locally under `/tmp`
Names will be:
```
/tmp/${prefix}-${cluster_name}-public-key.pub
/tmp/${prefix}-${cluster_name}-private-key.pem
```
Also we store the keys on key vault as secret:
To download keys from key vault use command:
```
az keyvault secret download --file private.pem --encoding utf-8 --vault-name  ${prefix}-${cluster_name} --name private-key --query "value"
az keyvault secret download --file private.pem --encoding utf-8 --vault-name  ${prefix}-${cluster_name} --name public-key --query "value"
```

## OBS
We support tiering to bucket.
In order to setup tiering, you must supply the following variables:
```hcl
set_obs_integration = true
obs_name            = "..."
obs_container_name  = "..."
blob_obs_access_key = "..."
```
In addition, you can supply (and override our default):
```hcl
tiering_ssd_percent = VALUE
```

## Clients
We support creating clients that will be mounted automatically to the cluster.
<br>In order to create clients you need to provide the number of clients you want (by default the number is 0),
for example:
```hcl
clients_number = 2
```
This will automatically create 2 clients.
<br>In addition you can supply these optional variables:
```hcl
client_instance_type = "Standard_D4_v4"
client_nics_num      = DESIRED_NUM
```
### Mounting clients in udp mode
In order to mount clients in udp mode you should pass the following param (in addition to the above):
```hcl
clients_use_dpdk = false
```

## NFS Protocol Gateways
We support creating NFS protocol gateways that will be mounted automatically to the cluster.
<br>In order to create you need to provide the number of protocol gateways instances you want (by default the number is 0),
for example:
```hcl
nfs_protocol_gateways_number = 1
```
This will automatically create 2 instances.
<br>In addition you can supply these optional variables:
```hcl
nfs_protocol_gateway_secondary_ips_per_nic = 3
nfs_protocol_gateway_instance_type         = "Standard_D8_v5"
nfs_protocol_gateway_nics_num              = 2
nfs_protocol_gateway_disk_size             = 48
nfs_protocol_gateway_frontend_cores_num    = 1
nfs_setup_protocol                         = false
```

<br>In order to create stateless clients, need to set variable:
```hcl
nfs_setup_protocol = true
```

## S3 Protocol Gateways
We support creating S3 protocol gateways that will be mounted automatically to the cluster.
<br>In order to create you need to provide the number of protocol gateways instances you want (by default the number is 0),

*The amount of S3 protocol gateways should be at least 3.*
</br>
for example:
```hcl
s3_protocol_gateways_number = 3
```
This will automatically create 3 instances.
<br>In addition you can supply these optional variables:
```hcl
s3_protocol_gateway_instance_type         = "Standard_D8_v5"
s3_protocol_gateway_nics_num              = 2
s3_protocol_gateway_disk_size             = 48
s3_protocol_gateway_frontend_cores_num    = 1
```

<br>In order to create stateless clients, need to set variable:
```hcl
s3_setup_protocol = true
```

## SMB Protocol Gateways
We support creating SMB protocol gateways that will be mounted automatically to the cluster.
<br>In order to create you need to provide the number of protocol gateways instances you want (by default the number is 0),

*The amount of SMB protocol gateways should be at least 3.*
</br>
for example:
```hcl
smb_protocol_gateways_number = 3
```
This will automatically create 2 instances.
<br>In addition you can supply these optional variables:
```hcl
smb_protocol_gateway_secondary_ips_per_nic = 3
smb_protocol_gateway_instance_type         = "Standard_D8_v5"
smb_protocol_gateway_nics_num              = 2
smb_protocol_gateway_disk_size             = 48
smb_protocol_gateway_frontend_cores_num    = 1
smb_setup_protocol                         = false
smb_cluster_name                           = ""
smb_domain_name                            = ""
smb_dns_ip_address                         = ""
```

<br>In order to create stateless clients, need to set variable:
```hcl
smb_setup_protocol = true
```

<br>To join an SMB cluster in Active Directory, need to pass domain username/password,
To join an SMB cluster in Active Directory, need to run manually command:

`weka smb domain join <smb_domain_username> <smb_domain_password> [--server smb_server_name]`.

<br>In order to enable SMBW, need to set variable:
```hcl
smbw_enabled = true
```

## Weka installation with proxy url
We support weka installation using custom proxy url.
```hcl
proxy_url = VALUE
```

<!-- BEGIN_TF_DOCS -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | >= 1.4.6 |
| <a name="requirement_azurerm"></a> [azurerm](#requirement\_azurerm) | ~>4.6.0 |
| <a name="requirement_local"></a> [local](#requirement\_local) | ~>2.4.0 |
| <a name="requirement_tls"></a> [tls](#requirement\_tls) | ~>4.0.4 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_azurerm"></a> [azurerm](#provider\_azurerm) | ~>4.6.0 |
| <a name="provider_local"></a> [local](#provider\_local) | ~>2.4.0 |
| <a name="provider_tls"></a> [tls](#provider\_tls) | ~>4.0.4 |

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_clients"></a> [clients](#module\_clients) | ./modules/clients | n/a |
| <a name="module_function_app_subnet_delegation"></a> [function\_app\_subnet\_delegation](#module\_function\_app\_subnet\_delegation) | ./modules/subnet_delegation | n/a |
| <a name="module_iam"></a> [iam](#module\_iam) | ./modules/iam | n/a |
| <a name="module_logic_app_subnet_delegation"></a> [logic\_app\_subnet\_delegation](#module\_logic\_app\_subnet\_delegation) | ./modules/subnet_delegation | n/a |
| <a name="module_logicapp"></a> [logicapp](#module\_logicapp) | ./modules/logic_app | n/a |
| <a name="module_network"></a> [network](#module\_network) | ./modules/network | n/a |
| <a name="module_nfs_protocol_gateways"></a> [nfs\_protocol\_gateways](#module\_nfs\_protocol\_gateways) | ./modules/protocol_gateways | n/a |
| <a name="module_peering"></a> [peering](#module\_peering) | ./modules/peering_vnets | n/a |
| <a name="module_s3_protocol_gateways"></a> [s3\_protocol\_gateways](#module\_s3\_protocol\_gateways) | ./modules/protocol_gateways | n/a |
| <a name="module_smb_protocol_gateways"></a> [smb\_protocol\_gateways](#module\_smb\_protocol\_gateways) | ./modules/protocol_gateways | n/a |

## Resources

| Name | Type |
|------|------|
| [azurerm_application_insights.application_insights](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/application_insights) | resource |
| [azurerm_key_vault.key_vault](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/key_vault) | resource |
| [azurerm_key_vault_access_policy.function_app_secret_permissions](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/key_vault_access_policy) | resource |
| [azurerm_key_vault_access_policy.key_vault_access_policy](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/key_vault_access_policy) | resource |
| [azurerm_key_vault_secret.function_app_default_key](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/key_vault_secret) | resource |
| [azurerm_key_vault_secret.get_weka_io_token](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/key_vault_secret) | resource |
| [azurerm_key_vault_secret.private_ssh_keys](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/key_vault_secret) | resource |
| [azurerm_key_vault_secret.public_ssh_keys](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/key_vault_secret) | resource |
| [azurerm_key_vault_secret.weka_deployment_password](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/key_vault_secret) | resource |
| [azurerm_key_vault_secret.weka_password_secret](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/key_vault_secret) | resource |
| [azurerm_lb.backend_lb](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/lb) | resource |
| [azurerm_lb.ui_lb](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/lb) | resource |
| [azurerm_lb_backend_address_pool.lb_backend_pool](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/lb_backend_address_pool) | resource |
| [azurerm_lb_backend_address_pool.ui_lb_backend_pool](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/lb_backend_address_pool) | resource |
| [azurerm_lb_probe.backend_lb_probe](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/lb_probe) | resource |
| [azurerm_lb_probe.ui_lb_probe](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/lb_probe) | resource |
| [azurerm_lb_rule.backend_lb_rule](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/lb_rule) | resource |
| [azurerm_lb_rule.ui_lb_rule](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/lb_rule) | resource |
| [azurerm_linux_function_app.function_app](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/linux_function_app) | resource |
| [azurerm_log_analytics_workspace.la_workspace](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/log_analytics_workspace) | resource |
| [azurerm_monitor_diagnostic_setting.function_diagnostic_setting](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/monitor_diagnostic_setting) | resource |
| [azurerm_monitor_diagnostic_setting.insights_diagnostic_setting](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/monitor_diagnostic_setting) | resource |
| [azurerm_private_dns_a_record.dns_a_record_backend_lb](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/private_dns_a_record) | resource |
| [azurerm_private_dns_resolver.dns_resolver](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/private_dns_resolver) | resource |
| [azurerm_private_dns_resolver_dns_forwarding_ruleset.dns_forwarding_ruleset](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/private_dns_resolver_dns_forwarding_ruleset) | resource |
| [azurerm_private_dns_resolver_forwarding_rule.resolver_forwarding_rule](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/private_dns_resolver_forwarding_rule) | resource |
| [azurerm_private_dns_resolver_outbound_endpoint.outbound_endpoint](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/private_dns_resolver_outbound_endpoint) | resource |
| [azurerm_private_dns_resolver_virtual_network_link.dns_forwarding_virtual_network_link](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/private_dns_resolver_virtual_network_link) | resource |
| [azurerm_private_dns_zone.blob](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/private_dns_zone) | resource |
| [azurerm_private_dns_zone.file](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/private_dns_zone) | resource |
| [azurerm_private_dns_zone_virtual_network_link.blob_privatelink](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/private_dns_zone_virtual_network_link) | resource |
| [azurerm_private_dns_zone_virtual_network_link.file_privatelink](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/private_dns_zone_virtual_network_link) | resource |
| [azurerm_private_endpoint.blob_endpoint](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/private_endpoint) | resource |
| [azurerm_private_endpoint.file_endpoint](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/private_endpoint) | resource |
| [azurerm_private_endpoint.weka_obs_blob_endpoint](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/private_endpoint) | resource |
| [azurerm_proximity_placement_group.ppg](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/proximity_placement_group) | resource |
| [azurerm_public_ip.backend_ip](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/public_ip) | resource |
| [azurerm_public_ip.ui_ip](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/public_ip) | resource |
| [azurerm_service_plan.app_service_plan](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/service_plan) | resource |
| [azurerm_storage_account.deployment_sa](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/storage_account) | resource |
| [azurerm_storage_account.logicapp](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/storage_account) | resource |
| [azurerm_storage_blob.nfs_state](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/storage_blob) | resource |
| [azurerm_storage_blob.state](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/storage_blob) | resource |
| [azurerm_storage_container.deployment](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/storage_container) | resource |
| [azurerm_storage_container.nfs_deployment](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/storage_container) | resource |
| [azurerm_storage_share.function_app_share](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/storage_share) | resource |
| [azurerm_subnet.dns_resolver_subnet](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/subnet) | resource |
| [local_file.private_key](https://registry.terraform.io/providers/hashicorp/local/latest/docs/resources/file) | resource |
| [local_file.public_key](https://registry.terraform.io/providers/hashicorp/local/latest/docs/resources/file) | resource |
| [tls_private_key.ssh_key](https://registry.terraform.io/providers/hashicorp/tls/latest/docs/resources/private_key) | resource |
| [azurerm_application_insights.application_insights](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/application_insights) | data source |
| [azurerm_client_config.current](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/client_config) | data source |
| [azurerm_function_app_host_keys.function_keys](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/function_app_host_keys) | data source |
| [azurerm_private_dns_zone.blob](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/private_dns_zone) | data source |
| [azurerm_resource_group.application_insights_rg](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/resource_group) | data source |
| [azurerm_resource_group.rg](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/resource_group) | data source |
| [azurerm_storage_account.deployment_blob](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/storage_account) | data source |
| [azurerm_storage_account.weka_obs](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/storage_account) | data source |
| [azurerm_storage_account_blob_container_sas.function_app_code_sas](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/storage_account_blob_container_sas) | data source |
| [azurerm_subnet.subnet](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/subnet) | data source |
| [azurerm_virtual_network.vnet](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/virtual_network) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_address_space"></a> [address\_space](#input\_address\_space) | The range of IP addresses the virtual network uses. | `string` | `"10.0.0.0/16"` | no |
| <a name="input_allow_ssh_cidrs"></a> [allow\_ssh\_cidrs](#input\_allow\_ssh\_cidrs) | Allow port 22, if not provided, i.e leaving the default empty list, the rule will not be included in the SG | `list(string)` | `[]` | no |
| <a name="input_allow_weka_api_cidrs"></a> [allow\_weka\_api\_cidrs](#input\_allow\_weka\_api\_cidrs) | Allow connection to port 14000 on weka backends from specified CIDRs, by default no CIDRs are allowed. All ports (including 14000) are allowed within Vnet | `list(string)` | `[]` | no |
| <a name="input_application_insights_instrumentation_key"></a> [application\_insights\_instrumentation\_key](#input\_application\_insights\_instrumentation\_key) | The Application Insights instrumentation key. | `string` | `""` | no |
| <a name="input_application_insights_name"></a> [application\_insights\_name](#input\_application\_insights\_name) | The Application Insights name. | `string` | `""` | no |
| <a name="input_application_insights_rg_name"></a> [application\_insights\_rg\_name](#input\_application\_insights\_rg\_name) | The Application Insights resource group name. | `string` | `""` | no |
| <a name="input_apt_repo_server"></a> [apt\_repo\_server](#input\_apt\_repo\_server) | The URL of the apt private repository. | `string` | `""` | no |
| <a name="input_assign_public_ip"></a> [assign\_public\_ip](#input\_assign\_public\_ip) | Determines whether to assign public IP to all instances deployed by TF module. Includes backends, clients and protocol gateways. | `string` | `"auto"` | no |
| <a name="input_backends_root_volume_size"></a> [backends\_root\_volume\_size](#input\_backends\_root\_volume\_size) | The backends' root disk size. | `number` | `null` | no |
| <a name="input_backends_weka_volume_size"></a> [backends\_weka\_volume\_size](#input\_backends\_weka\_volume\_size) | The default disk size. | `number` | `48` | no |
| <a name="input_client_arch"></a> [client\_arch](#input\_client\_arch) | Use arch for ami id, value can be arm64/x86\_64. | `string` | `null` | no |
| <a name="input_client_frontend_cores"></a> [client\_frontend\_cores](#input\_client\_frontend\_cores) | The client NICs number. | `number` | `1` | no |
| <a name="input_client_identity_name"></a> [client\_identity\_name](#input\_client\_identity\_name) | The user assigned identity name for the client instances (if empty - new one is created). | `string` | `""` | no |
| <a name="input_client_instance_type"></a> [client\_instance\_type](#input\_client\_instance\_type) | The client virtual machine type (sku) to deploy. | `string` | `""` | no |
| <a name="input_client_placement_group_id"></a> [client\_placement\_group\_id](#input\_client\_placement\_group\_id) | The client instances placement group id. Backend placement group can be reused. If not specified placement group will be created automatically | `string` | `""` | no |
| <a name="input_client_source_image_id"></a> [client\_source\_image\_id](#input\_client\_source\_image\_id) | Use weka custom image, ubuntu 20.04 with kernel 5.4 and ofed 5.8-1.1.2.1 / ubuntu arm 20.04 with kernel 5.4 and ofed 5.9-0.5.6.0 | `string` | `""` | no |
| <a name="input_clients_custom_data"></a> [clients\_custom\_data](#input\_clients\_custom\_data) | Custom data to pass to the client instances. Deprecated, use `clients_custom_data_post_mount` instead. | `string` | `""` | no |
| <a name="input_clients_custom_data_post_mount"></a> [clients\_custom\_data\_post\_mount](#input\_clients\_custom\_data\_post\_mount) | Custom data to pass to the client instances, will run after weka agent install and mount. | `string` | `""` | no |
| <a name="input_clients_custom_data_pre_mount"></a> [clients\_custom\_data\_pre\_mount](#input\_clients\_custom\_data\_pre\_mount) | Custom data to pass to the client instances, will run before weka agent install and mount. | `string` | `""` | no |
| <a name="input_clients_number"></a> [clients\_number](#input\_clients\_number) | The number of client virtual machines to deploy. | `number` | `0` | no |
| <a name="input_clients_root_volume_size"></a> [clients\_root\_volume\_size](#input\_clients\_root\_volume\_size) | The client's root volume size in GB | `number` | `null` | no |
| <a name="input_clients_use_dpdk"></a> [clients\_use\_dpdk](#input\_clients\_use\_dpdk) | Mount weka clients in DPDK mode | `bool` | `true` | no |
| <a name="input_clients_use_vmss"></a> [clients\_use\_vmss](#input\_clients\_use\_vmss) | Use VMSS for clients | `bool` | `false` | no |
| <a name="input_cluster_name"></a> [cluster\_name](#input\_cluster\_name) | Cluster name | `string` | `"poc"` | no |
| <a name="input_cluster_size"></a> [cluster\_size](#input\_cluster\_size) | The number of virtual machines to deploy. | `number` | `6` | no |
| <a name="input_clusterization_target"></a> [clusterization\_target](#input\_clusterization\_target) | The clusterization target | `number` | `null` | no |
| <a name="input_containers_config_map"></a> [containers\_config\_map](#input\_containers\_config\_map) | Maps the number of objects and memory size per machine type. | <pre>map(object({<br>    compute  = number<br>    drive    = number<br>    frontend = number<br>    nvme     = number<br>    nics     = number<br>    memory   = list(string)<br>  }))</pre> | <pre>{<br>  "Standard_L16as_v3": {<br>    "compute": 4,<br>    "drive": 2,<br>    "frontend": 1,<br>    "memory": [<br>      "72GB",<br>      "73GB"<br>    ],<br>    "nics": 8,<br>    "nvme": 2<br>  },<br>  "Standard_L16s_v3": {<br>    "compute": 4,<br>    "drive": 2,<br>    "frontend": 1,<br>    "memory": [<br>      "79GB",<br>      "72GB"<br>    ],<br>    "nics": 8,<br>    "nvme": 2<br>  },<br>  "Standard_L32as_v3": {<br>    "compute": 4,<br>    "drive": 2,<br>    "frontend": 1,<br>    "memory": [<br>      "190GB",<br>      "190GB"<br>    ],<br>    "nics": 8,<br>    "nvme": 4<br>  },<br>  "Standard_L32s_v3": {<br>    "compute": 4,<br>    "drive": 2,<br>    "frontend": 1,<br>    "memory": [<br>      "197GB",<br>      "189GB"<br>    ],<br>    "nics": 8,<br>    "nvme": 4<br>  },<br>  "Standard_L48as_v3": {<br>    "compute": 3,<br>    "drive": 3,<br>    "frontend": 1,<br>    "memory": [<br>      "308GB",<br>      "308GB"<br>    ],<br>    "nics": 8,<br>    "nvme": 6<br>  },<br>  "Standard_L48s_v3": {<br>    "compute": 3,<br>    "drive": 3,<br>    "frontend": 1,<br>    "memory": [<br>      "314GB",<br>      "306GB"<br>    ],<br>    "nics": 8,<br>    "nvme": 6<br>  },<br>  "Standard_L64as_v3": {<br>    "compute": 4,<br>    "drive": 2,<br>    "frontend": 1,<br>    "memory": [<br>      "384GB",<br>      "384GB"<br>    ],<br>    "nics": 8,<br>    "nvme": 8<br>  },<br>  "Standard_L64s_v3": {<br>    "compute": 4,<br>    "drive": 2,<br>    "frontend": 1,<br>    "memory": [<br>      "357GB",<br>      "384GB"<br>    ],<br>    "nics": 8,<br>    "nvme": 8<br>  },<br>  "Standard_L80as_v3": {<br>    "compute": 4,<br>    "drive": 2,<br>    "frontend": 1,<br>    "memory": [<br>      "384GB",<br>      "384GB"<br>    ],<br>    "nics": 8,<br>    "nvme": 8<br>  },<br>  "Standard_L80s_v3": {<br>    "compute": 4,<br>    "drive": 2,<br>    "frontend": 1,<br>    "memory": [<br>      "384GB",<br>      "384GB"<br>    ],<br>    "nics": 8,<br>    "nvme": 8<br>  },<br>  "Standard_L8as_v3": {<br>    "compute": 1,<br>    "drive": 1,<br>    "frontend": 1,<br>    "memory": [<br>      "29GB",<br>      "29GB"<br>    ],<br>    "nics": 4,<br>    "nvme": 1<br>  },<br>  "Standard_L8s_v3": {<br>    "compute": 1,<br>    "drive": 1,<br>    "frontend": 1,<br>    "memory": [<br>      "33GB",<br>      "31GB"<br>    ],<br>    "nics": 4,<br>    "nvme": 1<br>  }<br>}</pre> | no |
| <a name="input_create_lb"></a> [create\_lb](#input\_create\_lb) | Create backend and UI load balancers for weka cluster. | `bool` | `true` | no |
| <a name="input_create_nat_gateway"></a> [create\_nat\_gateway](#input\_create\_nat\_gateway) | NAT needs to be created when no public ip is assigned to the backend, to allow internet access | `bool` | `false` | no |
| <a name="input_create_storage_account_private_links"></a> [create\_storage\_account\_private\_links](#input\_create\_storage\_account\_private\_links) | Create private links for storage accounts (needed in case if public network access for the storage account is disabled). | `bool` | `false` | no |
| <a name="input_debug_down_backends_removal_timeout"></a> [debug\_down\_backends\_removal\_timeout](#input\_debug\_down\_backends\_removal\_timeout) | Don't change this value without consulting weka support team. Timeout for removing down backends. Valid time units are ns, us (or µs), ms, s, m, h. | `string` | `"3h"` | no |
| <a name="input_deployment_container_name"></a> [deployment\_container\_name](#input\_deployment\_container\_name) | Name of exising deployment container | `string` | `""` | no |
| <a name="input_deployment_file_share_name"></a> [deployment\_file\_share\_name](#input\_deployment\_file\_share\_name) | Name of exising deployment file share. Will use '<deployment\_storage\_account\_name>-share' name if not provided. | `string` | `""` | no |
| <a name="input_deployment_function_app_code_blob"></a> [deployment\_function\_app\_code\_blob](#input\_deployment\_function\_app\_code\_blob) | The path to the function app code blob file. | `string` | `""` | no |
| <a name="input_deployment_storage_account_name"></a> [deployment\_storage\_account\_name](#input\_deployment\_storage\_account\_name) | Name of exising deployment storage account | `string` | `""` | no |
| <a name="input_enable_application_insights"></a> [enable\_application\_insights](#input\_enable\_application\_insights) | Enable Application Insights. | `bool` | `true` | no |
| <a name="input_function_access_restriction_enabled"></a> [function\_access\_restriction\_enabled](#input\_function\_access\_restriction\_enabled) | Allow public access, Access restrictions apply to inbound access to internal vent | `bool` | `false` | no |
| <a name="input_function_app_dist"></a> [function\_app\_dist](#input\_function\_app\_dist) | Function app code dist | `string` | `"dev"` | no |
| <a name="input_function_app_identity_name"></a> [function\_app\_identity\_name](#input\_function\_app\_identity\_name) | The user assigned identity name for the function app (if empty - new one is created). | `string` | `""` | no |
| <a name="input_function_app_log_level"></a> [function\_app\_log\_level](#input\_function\_app\_log\_level) | Log level for function app (from -1 to 5). See https://github.com/rs/zerolog#leveled-logging | `number` | `1` | no |
| <a name="input_function_app_storage_account_container_prefix"></a> [function\_app\_storage\_account\_container\_prefix](#input\_function\_app\_storage\_account\_container\_prefix) | Weka storage account container name prefix | `string` | `"weka-tf-functions-deployment-"` | no |
| <a name="input_function_app_storage_account_prefix"></a> [function\_app\_storage\_account\_prefix](#input\_function\_app\_storage\_account\_prefix) | Weka storage account name prefix | `string` | `"weka"` | no |
| <a name="input_function_app_subnet_delegation_cidr"></a> [function\_app\_subnet\_delegation\_cidr](#input\_function\_app\_subnet\_delegation\_cidr) | Subnet delegation enables you to designate a specific subnet for an Azure PaaS service. | `string` | `"10.0.1.0/25"` | no |
| <a name="input_function_app_subnet_delegation_id"></a> [function\_app\_subnet\_delegation\_id](#input\_function\_app\_subnet\_delegation\_id) | Required to specify if subnet\_name were used to specify pre-defined subnets for weka. Function subnet delegation requires an additional subnet, and in the case of pre-defined networking this one also should be pre-defined | `string` | `""` | no |
| <a name="input_function_app_version"></a> [function\_app\_version](#input\_function\_app\_version) | Function app code version (hash) | `string` | `"d56b6c2fe62fdbc26fba3430df521913"` | no |
| <a name="input_get_weka_io_token"></a> [get\_weka\_io\_token](#input\_get\_weka\_io\_token) | The token to download the Weka release from get.weka.io. | `string` | `""` | no |
| <a name="input_hotspare"></a> [hotspare](#input\_hotspare) | Number of hotspares to set on weka cluster. Refer to https://docs.weka.io/weka-system-overview/ssd-capacity-management#hot-spare | `number` | `1` | no |
| <a name="input_install_cluster_dpdk"></a> [install\_cluster\_dpdk](#input\_install\_cluster\_dpdk) | Install weka cluster with DPDK | `bool` | `true` | no |
| <a name="input_install_weka_url"></a> [install\_weka\_url](#input\_install\_weka\_url) | The URL of the Weka release download tar file. | `string` | `""` | no |
| <a name="input_instance_type"></a> [instance\_type](#input\_instance\_type) | The virtual machine type (sku) to deploy. | `string` | `"Standard_L8s_v3"` | no |
| <a name="input_key_vault_purge_protection_enabled"></a> [key\_vault\_purge\_protection\_enabled](#input\_key\_vault\_purge\_protection\_enabled) | Enable purge protection for the key vault. | `bool` | `false` | no |
| <a name="input_log_analytics_workspace_id"></a> [log\_analytics\_workspace\_id](#input\_log\_analytics\_workspace\_id) | The Log Analytics workspace id. | `string` | `""` | no |
| <a name="input_logic_app_identity_name"></a> [logic\_app\_identity\_name](#input\_logic\_app\_identity\_name) | The user assigned identity name for the logic app (if empty - new one is created). | `string` | `""` | no |
| <a name="input_logic_app_subnet_delegation_cidr"></a> [logic\_app\_subnet\_delegation\_cidr](#input\_logic\_app\_subnet\_delegation\_cidr) | Subnet delegation enables you to designate a specific subnet for an Azure PaaS service. | `string` | `"10.0.3.0/25"` | no |
| <a name="input_logic_app_subnet_delegation_id"></a> [logic\_app\_subnet\_delegation\_id](#input\_logic\_app\_subnet\_delegation\_id) | Required to specify if subnet\_name were used to specify pre-defined subnets for weka. Logicapp subnet delegation requires an additional subnet, and in the case of pre-defined networking this one also should be pre-defined | `string` | `""` | no |
| <a name="input_nfs_deployment_container_name"></a> [nfs\_deployment\_container\_name](#input\_nfs\_deployment\_container\_name) | Name of exising protocol deployment container | `string` | `""` | no |
| <a name="input_nfs_interface_group_name"></a> [nfs\_interface\_group\_name](#input\_nfs\_interface\_group\_name) | Interface group name. | `string` | `"weka-ig"` | no |
| <a name="input_nfs_protocol_gateway_disk_size"></a> [nfs\_protocol\_gateway\_disk\_size](#input\_nfs\_protocol\_gateway\_disk\_size) | The protocol gateways' default disk size. | `number` | `48` | no |
| <a name="input_nfs_protocol_gateway_fe_cores_num"></a> [nfs\_protocol\_gateway\_fe\_cores\_num](#input\_nfs\_protocol\_gateway\_fe\_cores\_num) | The number of frontend cores on single protocol gateway machine. | `number` | `1` | no |
| <a name="input_nfs_protocol_gateway_instance_type"></a> [nfs\_protocol\_gateway\_instance\_type](#input\_nfs\_protocol\_gateway\_instance\_type) | The protocol gateways' virtual machine type (sku) to deploy. | `string` | `"Standard_D8_v5"` | no |
| <a name="input_nfs_protocol_gateway_secondary_ips_per_nic"></a> [nfs\_protocol\_gateway\_secondary\_ips\_per\_nic](#input\_nfs\_protocol\_gateway\_secondary\_ips\_per\_nic) | Number of secondary IPs per single NIC per protocol gateway virtual machine. | `number` | `0` | no |
| <a name="input_nfs_protocol_gateways_number"></a> [nfs\_protocol\_gateways\_number](#input\_nfs\_protocol\_gateways\_number) | The number of protocol gateway virtual machines to deploy. | `number` | `0` | no |
| <a name="input_nfs_setup_protocol"></a> [nfs\_setup\_protocol](#input\_nfs\_setup\_protocol) | Config protocol, default if false | `bool` | `false` | no |
| <a name="input_placement_group_id"></a> [placement\_group\_id](#input\_placement\_group\_id) | Proximity placement group to use for the vmss. If not passed, will be created automatically. | `string` | `""` | no |
| <a name="input_post_cluster_setup_script"></a> [post\_cluster\_setup\_script](#input\_post\_cluster\_setup\_script) | A script to run after the cluster is up | `string` | `""` | no |
| <a name="input_prefix"></a> [prefix](#input\_prefix) | Prefix for all resources | `string` | `"weka"` | no |
| <a name="input_private_dns_rg_name"></a> [private\_dns\_rg\_name](#input\_private\_dns\_rg\_name) | The private DNS zone resource group name. Required when private\_dns\_zone\_name is set. | `string` | `""` | no |
| <a name="input_private_dns_zone_name"></a> [private\_dns\_zone\_name](#input\_private\_dns\_zone\_name) | The private DNS zone name. | `string` | `""` | no |
| <a name="input_private_dns_zone_use"></a> [private\_dns\_zone\_use](#input\_private\_dns\_zone\_use) | Determines whether to use private DNS zone. Required for LB record creation. | `bool` | `true` | no |
| <a name="input_protection_level"></a> [protection\_level](#input\_protection\_level) | Cluster data protection level. | `number` | `2` | no |
| <a name="input_protocol_gateways_identity_name"></a> [protocol\_gateways\_identity\_name](#input\_protocol\_gateways\_identity\_name) | The user assigned identity name for the protocol gateways instances (if empty - new one is created). | `string` | `""` | no |
| <a name="input_proxy_url"></a> [proxy\_url](#input\_proxy\_url) | Weka home proxy url | `string` | `""` | no |
| <a name="input_read_function_zip_from_storage_account"></a> [read\_function\_zip\_from\_storage\_account](#input\_read\_function\_zip\_from\_storage\_account) | Read function app zip from storage account (is read from public distribution storage account by default). | `bool` | `false` | no |
| <a name="input_rg_name"></a> [rg\_name](#input\_rg\_name) | A predefined resource group in the Azure subscription. | `string` | n/a | yes |
| <a name="input_s3_protocol_gateway_disk_size"></a> [s3\_protocol\_gateway\_disk\_size](#input\_s3\_protocol\_gateway\_disk\_size) | The protocol gateways' default disk size. | `number` | `48` | no |
| <a name="input_s3_protocol_gateway_fe_cores_num"></a> [s3\_protocol\_gateway\_fe\_cores\_num](#input\_s3\_protocol\_gateway\_fe\_cores\_num) | The number of frontend cores on single protocol gateway machine. | `number` | `1` | no |
| <a name="input_s3_protocol_gateway_instance_type"></a> [s3\_protocol\_gateway\_instance\_type](#input\_s3\_protocol\_gateway\_instance\_type) | The protocol gateways' virtual machine type (sku) to deploy. | `string` | `"Standard_D8_v5"` | no |
| <a name="input_s3_protocol_gateways_number"></a> [s3\_protocol\_gateways\_number](#input\_s3\_protocol\_gateways\_number) | The number of protocol gateway virtual machines to deploy. | `number` | `0` | no |
| <a name="input_s3_setup_protocol"></a> [s3\_setup\_protocol](#input\_s3\_setup\_protocol) | Config protocol, default if false | `bool` | `false` | no |
| <a name="input_script_post_cluster_creation"></a> [script\_post\_cluster\_creation](#input\_script\_post\_cluster\_creation) | Script to run after cluster creation | `string` | `""` | no |
| <a name="input_script_pre_start_io"></a> [script\_pre\_start\_io](#input\_script\_pre\_start\_io) | Script to run before starting IO | `string` | `""` | no |
| <a name="input_set_dedicated_fe_container"></a> [set\_dedicated\_fe\_container](#input\_set\_dedicated\_fe\_container) | Create cluster with FE containers | `bool` | `false` | no |
| <a name="input_set_default_fs"></a> [set\_default\_fs](#input\_set\_default\_fs) | Set the default filesystem which will use the full available capacity | `bool` | `true` | no |
| <a name="input_sg_custom_ingress_rules"></a> [sg\_custom\_ingress\_rules](#input\_sg\_custom\_ingress\_rules) | Custom inbound rules to be added to the security group. | <pre>list(object({<br>    from_port  = string<br>    to_port    = string<br>    protocol   = string<br>    cidr_block = string<br>  }))</pre> | `[]` | no |
| <a name="input_sg_id"></a> [sg\_id](#input\_sg\_id) | The security group id. | `string` | `""` | no |
| <a name="input_smb_cluster_name"></a> [smb\_cluster\_name](#input\_smb\_cluster\_name) | The name of the SMB setup. | `string` | `"Weka-SMB"` | no |
| <a name="input_smb_create_private_dns_resolver"></a> [smb\_create\_private\_dns\_resolver](#input\_smb\_create\_private\_dns\_resolver) | Create dns resolver for smb with outbound rule | `bool` | `false` | no |
| <a name="input_smb_dns_ip_address"></a> [smb\_dns\_ip\_address](#input\_smb\_dns\_ip\_address) | DNS IP address | `string` | `""` | no |
| <a name="input_smb_dns_resolver_subnet_delegation_cidr"></a> [smb\_dns\_resolver\_subnet\_delegation\_cidr](#input\_smb\_dns\_resolver\_subnet\_delegation\_cidr) | Cidr of dns resolver of subnet, for SMB | `string` | `"10.0.4.0/28"` | no |
| <a name="input_smb_dns_resolver_subnet_delegation_id"></a> [smb\_dns\_resolver\_subnet\_delegation\_id](#input\_smb\_dns\_resolver\_subnet\_delegation\_id) | Required to specify if subnet\_id were used to specify pre-defined for SMB dns resolver subnet, requires an additional subnet, '/subscriptions/../resourceGroups/../providers/Microsoft.Network/virtualNetworks/../subnets/..' | `string` | `""` | no |
| <a name="input_smb_domain_name"></a> [smb\_domain\_name](#input\_smb\_domain\_name) | The domain to join the SMB cluster to. | `string` | `""` | no |
| <a name="input_smb_protocol_gateway_disk_size"></a> [smb\_protocol\_gateway\_disk\_size](#input\_smb\_protocol\_gateway\_disk\_size) | The protocol gateways' default disk size. | `number` | `48` | no |
| <a name="input_smb_protocol_gateway_fe_cores_num"></a> [smb\_protocol\_gateway\_fe\_cores\_num](#input\_smb\_protocol\_gateway\_fe\_cores\_num) | The number of frontend cores on single protocol gateway machine. | `number` | `1` | no |
| <a name="input_smb_protocol_gateway_instance_type"></a> [smb\_protocol\_gateway\_instance\_type](#input\_smb\_protocol\_gateway\_instance\_type) | The protocol gateways' virtual machine type (sku) to deploy. | `string` | `"Standard_D8_v5"` | no |
| <a name="input_smb_protocol_gateway_secondary_ips_per_nic"></a> [smb\_protocol\_gateway\_secondary\_ips\_per\_nic](#input\_smb\_protocol\_gateway\_secondary\_ips\_per\_nic) | Number of secondary IPs per single NIC per protocol gateway virtual machine. | `number` | `0` | no |
| <a name="input_smb_protocol_gateways_number"></a> [smb\_protocol\_gateways\_number](#input\_smb\_protocol\_gateways\_number) | The number of protocol gateway virtual machines to deploy. | `number` | `0` | no |
| <a name="input_smb_setup_protocol"></a> [smb\_setup\_protocol](#input\_smb\_setup\_protocol) | Config protocol, default if false | `bool` | `false` | no |
| <a name="input_smbw_enabled"></a> [smbw\_enabled](#input\_smbw\_enabled) | Enable SMBW protocol. This option should be provided before cluster is created to leave extra capacity for SMBW setup. | `bool` | `true` | no |
| <a name="input_source_image_id"></a> [source\_image\_id](#input\_source\_image\_id) | Use weka custom image, ubuntu 20.04 with kernel 5.4 and ofed 5.8-1.1.2.1 | `string` | `"/communityGalleries/WekaIO-ddbef83d-dec1-42d0-998a-3c083f1450b7/images/weka_custom_image/versions/1.0.1"` | no |
| <a name="input_ssh_public_key"></a> [ssh\_public\_key](#input\_ssh\_public\_key) | Ssh public key to pass to vms. | `string` | `null` | no |
| <a name="input_storage_account_allowed_ips"></a> [storage\_account\_allowed\_ips](#input\_storage\_account\_allowed\_ips) | IP ranges to allow access from the internet or your on-premises networks to storage accounts. | `list(string)` | `[]` | no |
| <a name="input_storage_account_public_network_access"></a> [storage\_account\_public\_network\_access](#input\_storage\_account\_public\_network\_access) | Public network access to the storage accounts. | `string` | `"Enabled"` | no |
| <a name="input_storage_blob_private_dns_zone_name"></a> [storage\_blob\_private\_dns\_zone\_name](#input\_storage\_blob\_private\_dns\_zone\_name) | The private DNS zone name for the storage account (blob). | `string` | `"privatelink.blob.core.windows.net"` | no |
| <a name="input_stripe_width"></a> [stripe\_width](#input\_stripe\_width) | Stripe width = cluster\_size - protection\_level - 1 (by default). | `number` | `-1` | no |
| <a name="input_subnet_name"></a> [subnet\_name](#input\_subnet\_name) | The subnet name. | `string` | `""` | no |
| <a name="input_subnet_prefix"></a> [subnet\_prefix](#input\_subnet\_prefix) | Address prefixes to use for the subnet | `string` | `"10.0.2.0/24"` | no |
| <a name="input_subscription_id"></a> [subscription\_id](#input\_subscription\_id) | The subscription id for the deployment. | `string` | n/a | yes |
| <a name="input_tags_map"></a> [tags\_map](#input\_tags\_map) | A map of tags to assign the same metadata to all resources in the environment. Format: key:value. | `map(string)` | `{}` | no |
| <a name="input_tiering_blob_obs_access_key"></a> [tiering\_blob\_obs\_access\_key](#input\_tiering\_blob\_obs\_access\_key) | The access key of the existing Blob object store container. If not provided, new obs will be created with given name (tiering\_obs\_name). | `string` | `""` | no |
| <a name="input_tiering_enable_obs_integration"></a> [tiering\_enable\_obs\_integration](#input\_tiering\_enable\_obs\_integration) | Determines whether to enable object stores integration with the Weka cluster. Set true to enable the integration. | `bool` | `false` | no |
| <a name="input_tiering_enable_ssd_percent"></a> [tiering\_enable\_ssd\_percent](#input\_tiering\_enable\_ssd\_percent) | When set\_obs\_integration is true, this variable sets the capacity percentage of the filesystem that resides on SSD. For example, for an SSD with a total capacity of 20GB, and the tiering\_ssd\_percent is set to 20, the total available capacity is 100GB. | `number` | `20` | no |
| <a name="input_tiering_obs_container_name"></a> [tiering\_obs\_container\_name](#input\_tiering\_obs\_container\_name) | Name of existing obs container name. | `string` | `""` | no |
| <a name="input_tiering_obs_name"></a> [tiering\_obs\_name](#input\_tiering\_obs\_name) | Name of existing obs storage account | `string` | `""` | no |
| <a name="input_tiering_obs_start_demote"></a> [tiering\_obs\_start\_demote](#input\_tiering\_obs\_start\_demote) | Target tiering cue (in seconds) before starting upload data to OBS (turning it into read cache). Default is 10 seconds. | `number` | `10` | no |
| <a name="input_tiering_obs_target_ssd_retention"></a> [tiering\_obs\_target\_ssd\_retention](#input\_tiering\_obs\_target\_ssd\_retention) | Target retention period (in seconds) before tiering to OBS (how long data will stay in SSD). Default is 86400 seconds (24 hours). | `number` | `86400` | no |
| <a name="input_traces_per_ionode"></a> [traces\_per\_ionode](#input\_traces\_per\_ionode) | The number of traces per ionode. Traces are low-level events generated by Weka processes and are used as troubleshooting information for support purposes. | `number` | `10` | no |
| <a name="input_user_data"></a> [user\_data](#input\_user\_data) | User data to pass to vms. | `string` | `""` | no |
| <a name="input_vm_username"></a> [vm\_username](#input\_vm\_username) | Provided as part of output for automated use of terraform, in case of custom AMI and automated use of outputs replace this with user that should be used for ssh connection | `string` | `"weka"` | no |
| <a name="input_vmss_identity_name"></a> [vmss\_identity\_name](#input\_vmss\_identity\_name) | The user assigned identity name for the vmss instances (if empty - new one is created). | `string` | `""` | no |
| <a name="input_vmss_single_placement_group"></a> [vmss\_single\_placement\_group](#input\_vmss\_single\_placement\_group) | Sets single\_placement\_group option for vmss. If true, a scale set is composed of a single placement group, and has a range of 0-100 VMs. | `bool` | `true` | no |
| <a name="input_vnet_name"></a> [vnet\_name](#input\_vnet\_name) | The virtual network name. | `string` | `""` | no |
| <a name="input_vnet_rg_name"></a> [vnet\_rg\_name](#input\_vnet\_rg\_name) | Resource group name of vnet. Will be used when vnet\_name is not provided. | `string` | `""` | no |
| <a name="input_vnets_to_peer_to_deployment_vnet"></a> [vnets\_to\_peer\_to\_deployment\_vnet](#input\_vnets\_to\_peer\_to\_deployment\_vnet) | List of vent-name:resource-group-name to peer | <pre>list(object({<br>    vnet = string<br>    rg   = string<br>  }))</pre> | `[]` | no |
| <a name="input_weka_home_url"></a> [weka\_home\_url](#input\_weka\_home\_url) | Weka Home url | `string` | `""` | no |
| <a name="input_weka_tar_storage_account_id"></a> [weka\_tar\_storage\_account\_id](#input\_weka\_tar\_storage\_account\_id) | ### private blob | `string` | `""` | no |
| <a name="input_weka_version"></a> [weka\_version](#input\_weka\_version) | The Weka version to deploy. | `string` | `""` | no |
| <a name="input_zone"></a> [zone](#input\_zone) | The zone in which the resources should be created. | `string` | `null` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_backend_ips"></a> [backend\_ips](#output\_backend\_ips) | If 'assign\_public\_ip' is set to true, it will output the public ips, If no it will output the private ips |
| <a name="output_backend_lb_private_ip"></a> [backend\_lb\_private\_ip](#output\_backend\_lb\_private\_ip) | Backend load balancer ip address |
| <a name="output_client_ips"></a> [client\_ips](#output\_client\_ips) | If 'private\_network' is set to false, it will output clients public ips, otherwise private ips. |
| <a name="output_client_vmss_ips"></a> [client\_vmss\_ips](#output\_client\_vmss\_ips) | If 'private\_network' is set to false, it will output clients public ips, otherwise private ips. |
| <a name="output_clients_vmss_name"></a> [clients\_vmss\_name](#output\_clients\_vmss\_name) | n/a |
| <a name="output_cluster_helper_commands"></a> [cluster\_helper\_commands](#output\_cluster\_helper\_commands) | Useful commands and script to interact with weka cluster |
| <a name="output_function_app_name"></a> [function\_app\_name](#output\_function\_app\_name) | Function app name |
| <a name="output_function_key_name"></a> [function\_key\_name](#output\_function\_key\_name) | Function app key name |
| <a name="output_functions_url"></a> [functions\_url](#output\_functions\_url) | Functions url and body for api request |
| <a name="output_key_vault_name"></a> [key\_vault\_name](#output\_key\_vault\_name) | Keyault name |
| <a name="output_nfs_vmss_name"></a> [nfs\_vmss\_name](#output\_nfs\_vmss\_name) | NFS protocol gateway vmss name |
| <a name="output_ppg_id"></a> [ppg\_id](#output\_ppg\_id) | Placement proximity group id |
| <a name="output_private_ssh_key"></a> [private\_ssh\_key](#output\_private\_ssh\_key) | If 'ssh\_public\_key' is set to null and no file provided, it will output the private ssh key location. |
| <a name="output_s3_protocol_gateway_ips"></a> [s3\_protocol\_gateway\_ips](#output\_s3\_protocol\_gateway\_ips) | If 'private\_network' is set to false, it will output smb protocol gateway public ips, otherwise private ips. |
| <a name="output_sg_id"></a> [sg\_id](#output\_sg\_id) | Security group id |
| <a name="output_smb_protocol_gateway_ips"></a> [smb\_protocol\_gateway\_ips](#output\_smb\_protocol\_gateway\_ips) | If 'private\_network' is set to false, it will output smb protocol gateway public ips, otherwise private ips. |
| <a name="output_subnet_name"></a> [subnet\_name](#output\_subnet\_name) | Subnet name |
| <a name="output_vm_username"></a> [vm\_username](#output\_vm\_username) | Provided as part of output for automated use of terraform, ssh user to weka cluster vm |
| <a name="output_vmss_name"></a> [vmss\_name](#output\_vmss\_name) | n/a |
| <a name="output_vnet_name"></a> [vnet\_name](#output\_vnet\_name) | Virtual network name |
| <a name="output_vnet_rg_name"></a> [vnet\_rg\_name](#output\_vnet\_rg\_name) | Virtual network resource group name |
| <a name="output_weka_cluster_admin_password_secret_name"></a> [weka\_cluster\_admin\_password\_secret\_name](#output\_weka\_cluster\_admin\_password\_secret\_name) | Weka cluster admin password secret name |
<!-- END_TF_DOCS -->
