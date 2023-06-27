# Azure-weka deployment Terraform package
The Weka cluster on Azure provides a fast and scalable platform to run, for example, performance-intensive applications and hybrid cloud workflows. It can also be used for object stores, tiering, and snapshots using the Azure Blob service.
The provided Azure-Weka Terraform package contains modules and examples you can customize according to your deployment needs. The installation is based on applying the customized Terraform variables file to a predefined Azure subscription. 
Applying the Terraform variables file performs the following:
- Creates resources in a predefined resource group, such as virtual machines, network interfaces, function app, load balancer, and more.
- Deploys Azure virtual machines.
- Installs the Weka software.
- Configures the Weka cluster.

<br> You can find [here](https://github.com/weka/terraform-azure-weka-essential) our essential deployment which creates only vms and placement group.

## Usage
```hcl
module "deploy-weka" {
   source                = "weka/weka/azure"
   version               = "3.0.1"
   prefix                = "weka"
   rg_name               = "myResourceGroup"
   vnet_name             = "weka-vpc-0"
   vnet_rg_name          = "myVnetResourceGroup"
   subnet_name           = "weka-subnet-0"
   sg_id                 = "security-group-id"
   get_weka_io_token     = "get_weka_io_token"
   cluster_name          = "myCluster"
   subnet_delegation      = "10.0.1.0/25"
   set_obs_integration   = false
   instance_type         = "Standard_L8s_v3"
   cluster_size          = 6
   subscription_id       = "mySubscriptionId"
   private_dns_zone_name = "myDns.private.net"
}
```

## Weka custom image
As you can see via `source_image_id` variable, we use our own custom image.
This is a community image that we created and uploaded to azure.
In case you would like to view how we created the image you can find it [here](https://github.com/weka/terraform-azure-weka-custom-image).
You can as well create it on your own subscription and use it.


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
client_nics_num = DESIRED_NUM
```
### Mounting clients in udp mode
In order to mount clients in udp mode you should pass the following param (in addition to the above):
```hcl
mount_clients_dpdk = false
```

<!-- BEGIN_TF_DOCS -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_azurerm"></a> [azurerm](#requirement\_azurerm) | ~>3.43.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_azurerm"></a> [azurerm](#provider\_azurerm) | ~>3.43.0 |
| <a name="provider_local"></a> [local](#provider\_local) | n/a |
| <a name="provider_null"></a> [null](#provider\_null) | n/a |
| <a name="provider_random"></a> [random](#provider\_random) | n/a |
| <a name="provider_tls"></a> [tls](#provider\_tls) | n/a |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [azurerm_application_insights.application_insights](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/application_insights) | resource |
| [azurerm_key_vault.key_vault](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/key_vault) | resource |
| [azurerm_key_vault_access_policy.function-app-get-secret-permission](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/key_vault_access_policy) | resource |
| [azurerm_key_vault_access_policy.key_vault_access_policy](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/key_vault_access_policy) | resource |
| [azurerm_key_vault_access_policy.logic-app-get-secret-permission](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/key_vault_access_policy) | resource |
| [azurerm_key_vault_access_policy.scale-up-logic-app-get-secret-permission](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/key_vault_access_policy) | resource |
| [azurerm_key_vault_secret.function_app_default_key](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/key_vault_secret) | resource |
| [azurerm_key_vault_secret.get_weka_io_token](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/key_vault_secret) | resource |
| [azurerm_key_vault_secret.private-ssh-keys](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/key_vault_secret) | resource |
| [azurerm_key_vault_secret.public-ssh-keys](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/key_vault_secret) | resource |
| [azurerm_key_vault_secret.weka_password_secret](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/key_vault_secret) | resource |
| [azurerm_lb.backend-lb](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/lb) | resource |
| [azurerm_lb.ui_lb](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/lb) | resource |
| [azurerm_lb_backend_address_pool.lb_backend_pool](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/lb_backend_address_pool) | resource |
| [azurerm_lb_backend_address_pool.ui_lb_backend_pool](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/lb_backend_address_pool) | resource |
| [azurerm_lb_probe.backend_lb_probe](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/lb_probe) | resource |
| [azurerm_lb_probe.ui_lb_probe](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/lb_probe) | resource |
| [azurerm_lb_rule.backend_lb_rule](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/lb_rule) | resource |
| [azurerm_lb_rule.ui_lb_rule](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/lb_rule) | resource |
| [azurerm_linux_function_app.function_app](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/linux_function_app) | resource |
| [azurerm_linux_virtual_machine_scale_set.vmss](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/linux_virtual_machine_scale_set) | resource |
| [azurerm_log_analytics_workspace.la_workspace](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/log_analytics_workspace) | resource |
| [azurerm_logic_app_action_custom.logic_app_action_fetch](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/logic_app_action_custom) | resource |
| [azurerm_logic_app_action_custom.logic_app_action_scale_down](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/logic_app_action_custom) | resource |
| [azurerm_logic_app_action_custom.logic_app_action_scale_up](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/logic_app_action_custom) | resource |
| [azurerm_logic_app_action_custom.logic_app_action_terminate](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/logic_app_action_custom) | resource |
| [azurerm_logic_app_action_custom.logic_app_action_transient](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/logic_app_action_custom) | resource |
| [azurerm_logic_app_action_custom.scale_down_logic_app_action_get_secret](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/logic_app_action_custom) | resource |
| [azurerm_logic_app_action_custom.scale_up_logic_app_action_get_secret](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/logic_app_action_custom) | resource |
| [azurerm_monitor_diagnostic_setting.function_diagnostic_setting](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/monitor_diagnostic_setting) | resource |
| [azurerm_monitor_diagnostic_setting.insights_diagnostic_setting](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/monitor_diagnostic_setting) | resource |
| [azurerm_monitor_diagnostic_setting.logic_app_diagnostic_setting](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/monitor_diagnostic_setting) | resource |
| [azurerm_monitor_diagnostic_setting.scale_up_logic_app_diagnostic_setting](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/monitor_diagnostic_setting) | resource |
| [azurerm_private_dns_a_record.dns_a_record_backend_lb](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/private_dns_a_record) | resource |
| [azurerm_proximity_placement_group.ppg](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/proximity_placement_group) | resource |
| [azurerm_resource_group_template_deployment.api_connections_template_deployment](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/resource_group_template_deployment) | resource |
| [azurerm_resource_group_template_deployment.workflow_scale_down_template_deployment](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/resource_group_template_deployment) | resource |
| [azurerm_resource_group_template_deployment.workflow_scale_up_template_deployment](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/resource_group_template_deployment) | resource |
| [azurerm_role_assignment.function-app-key-user-access-admin](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_role_assignment.function-app-key-vault-secrets-user](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_role_assignment.function-app-reader](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_role_assignment.function-app-scale-set-machine-owner](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_role_assignment.function-assignment](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_role_assignment.logic-app-key-vault-secrets-user](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_role_assignment.logic-app-storage-account-contributor](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_role_assignment.scale-up-logic-app-key-vault-secrets-user](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_role_assignment.scale-up-logic-app-storage-account-contributor](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_role_assignment.storage-account-contributor](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_role_assignment.storage-blob-data-owner](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_role_assignment.vm_role_assignment](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_service_plan.app_service_plan](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/service_plan) | resource |
| [azurerm_storage_account.deployment_sa](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/storage_account) | resource |
| [azurerm_storage_blob.state](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/storage_blob) | resource |
| [azurerm_storage_container.deployment](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/storage_container) | resource |
| [azurerm_subnet.subnet-delegation](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/subnet) | resource |
| [local_file.private_key](https://registry.terraform.io/providers/hashicorp/local/latest/docs/resources/file) | resource |
| [local_file.public_key](https://registry.terraform.io/providers/hashicorp/local/latest/docs/resources/file) | resource |
| [null_resource.force-delete-vmss](https://registry.terraform.io/providers/hashicorp/null/latest/docs/resources/resource) | resource |
| [random_password.weka_password](https://registry.terraform.io/providers/hashicorp/random/latest/docs/resources/password) | resource |
| [tls_private_key.ssh_key](https://registry.terraform.io/providers/hashicorp/tls/latest/docs/resources/private_key) | resource |
| [azurerm_client_config.current](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/client_config) | data source |
| [azurerm_function_app_host_keys.function_keys](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/function_app_host_keys) | data source |
| [azurerm_resource_group.rg](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/resource_group) | data source |
| [azurerm_subnet.subnet](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/subnet) | data source |
| [azurerm_subscription.primary](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/subscription) | data source |
| [azurerm_virtual_network.vnet](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/virtual_network) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_add_frontend_containers"></a> [add\_frontend\_containers](#input\_add\_frontend\_containers) | Create cluster with FE containers | `bool` | `true` | no |
| <a name="input_apt_repo_url"></a> [apt\_repo\_url](#input\_apt\_repo\_url) | The URL of the apt private repository. | `string` | `""` | no |
| <a name="input_blob_obs_access_key"></a> [blob\_obs\_access\_key](#input\_blob\_obs\_access\_key) | The access key of the existing Blob object store container. | `string` | `""` | no |
| <a name="input_cluster_name"></a> [cluster\_name](#input\_cluster\_name) | The cluster name. | `string` | `"poc"` | no |
| <a name="input_cluster_size"></a> [cluster\_size](#input\_cluster\_size) | The number of virtual machines to deploy. | `number` | `6` | no |
| <a name="input_container_number_map"></a> [container\_number\_map](#input\_container\_number\_map) | Maps the number of objects and memory size per machine type. | <pre>map(object({<br>    compute  = number<br>    drive    = number<br>    frontend = number<br>    nvme     = number<br>    nics     = number<br>    memory   = list(string)<br>  }))</pre> | <pre>{<br>  "Standard_L16s_v3": {<br>    "compute": 4,<br>    "drive": 2,<br>    "frontend": 1,<br>    "memory": [<br>      "79GB",<br>      "72GB"<br>    ],<br>    "nics": 8,<br>    "nvme": 2<br>  },<br>  "Standard_L32s_v3": {<br>    "compute": 4,<br>    "drive": 2,<br>    "frontend": 1,<br>    "memory": [<br>      "197GB",<br>      "189GB"<br>    ],<br>    "nics": 8,<br>    "nvme": 4<br>  },<br>  "Standard_L48s_v3": {<br>    "compute": 3,<br>    "drive": 3,<br>    "frontend": 1,<br>    "memory": [<br>      "314GB",<br>      "306GB"<br>    ],<br>    "nics": 8,<br>    "nvme": 6<br>  },<br>  "Standard_L64s_v3": {<br>    "compute": 4,<br>    "drive": 2,<br>    "frontend": 1,<br>    "memory": [<br>      "357GB",<br>      "418GB"<br>    ],<br>    "nics": 8,<br>    "nvme": 8<br>  },<br>  "Standard_L8s_v3": {<br>    "compute": 1,<br>    "drive": 1,<br>    "frontend": 1,<br>    "memory": [<br>      "33GB",<br>      "31GB"<br>    ],<br>    "nics": 4,<br>    "nvme": 1<br>  }<br>}</pre> | no |
| <a name="input_default_disk_size"></a> [default\_disk\_size](#input\_default\_disk\_size) | The default disk size. | `number` | `48` | no |
| <a name="input_function_app_dist"></a> [function\_app\_dist](#input\_function\_app\_dist) | Function app code dist | `string` | `"dev"` | no |
| <a name="input_function_app_log_level"></a> [function\_app\_log\_level](#input\_function\_app\_log\_level) | Log level for function app (from -1 to 5). See https://github.com/rs/zerolog#leveled-logging | `number` | `1` | no |
| <a name="input_function_app_storage_account_container_prefix"></a> [function\_app\_storage\_account\_container\_prefix](#input\_function\_app\_storage\_account\_container\_prefix) | Weka storage account container name prefix | `string` | `"weka-tf-functions-deployment-"` | no |
| <a name="input_function_app_storage_account_prefix"></a> [function\_app\_storage\_account\_prefix](#input\_function\_app\_storage\_account\_prefix) | Weka storage account name prefix | `string` | `"weka"` | no |
| <a name="input_function_app_version"></a> [function\_app\_version](#input\_function\_app\_version) | Function app code version (hash) | `string` | `"5e6314a36e7a5a2c017798ee96d6e884"` | no |
| <a name="input_get_weka_io_token"></a> [get\_weka\_io\_token](#input\_get\_weka\_io\_token) | The token to download the Weka release from get.weka.io. | `string` | `""` | no |
| <a name="input_hotspare"></a> [hotspare](#input\_hotspare) | Hot-spare value. | `number` | `1` | no |
| <a name="input_install_cluster_dpdk"></a> [install\_cluster\_dpdk](#input\_install\_cluster\_dpdk) | Install weka cluster with DPDK | `bool` | `true` | no |
| <a name="input_install_weka_url"></a> [install\_weka\_url](#input\_install\_weka\_url) | The URL of the Weka release download tar file. | `string` | `""` | no |
| <a name="input_instance_type"></a> [instance\_type](#input\_instance\_type) | The virtual machine type (sku) to deploy. | `string` | `"Standard_L8s_v3"` | no |
| <a name="input_obs_container_name"></a> [obs\_container\_name](#input\_obs\_container\_name) | Name of existing obs conatiner name | `string` | `""` | no |
| <a name="input_obs_name"></a> [obs\_name](#input\_obs\_name) | Name of existing obs storage account | `string` | `""` | no |
| <a name="input_prefix"></a> [prefix](#input\_prefix) | The prefix for all the resource names. For example, the prefix for your system name. | `string` | `"weka"` | no |
| <a name="input_private_dns_zone_name"></a> [private\_dns\_zone\_name](#input\_private\_dns\_zone\_name) | The private DNS zone name. | `string` | `null` | no |
| <a name="input_private_network"></a> [private\_network](#input\_private\_network) | Determines whether to enable a private or public network. The default is public network. | `bool` | `false` | no |
| <a name="input_protection_level"></a> [protection\_level](#input\_protection\_level) | Cluster data protection level. | `number` | `2` | no |
| <a name="input_rg_name"></a> [rg\_name](#input\_rg\_name) | A predefined resource group in the Azure subscription. | `string` | n/a | yes |
| <a name="input_set_obs_integration"></a> [set\_obs\_integration](#input\_set\_obs\_integration) | Determines whether to enable object stores integration with the Weka cluster. Set true to enable the integration. | `bool` | `false` | no |
| <a name="input_sg_id"></a> [sg\_id](#input\_sg\_id) | The security group id. | `string` | n/a | yes |
| <a name="input_source_image_id"></a> [source\_image\_id](#input\_source\_image\_id) | Use weka custom image, ubuntu 20.04 with kernel 5.4 and ofed 5.8-1.1.2.1 | `string` | `"/communityGalleries/WekaIO-d7d3f308-d5a1-4c45-8e8a-818aed57375a/images/ubuntu20.04/versions/latest"` | no |
| <a name="input_ssh_private_key"></a> [ssh\_private\_key](#input\_ssh\_private\_key) | The path to the VM private key. If it is not set, the key is auto-generated. If it is set, also set the ssh\_private\_key. The private key used for connecting to the deployed virtual machines to initiate the clusterization of Weka. | `string` | `null` | no |
| <a name="input_ssh_public_key"></a> [ssh\_public\_key](#input\_ssh\_public\_key) | The path to the VM public key. If it is not set, the key is auto-generated. If it is set, also set the ssh\_private\_key. | `string` | `null` | no |
| <a name="input_stripe_width"></a> [stripe\_width](#input\_stripe\_width) | Stripe width = cluster\_size - protection\_level - 1 (by default). | `number` | `-1` | no |
| <a name="input_subnet_delegation"></a> [subnet\_delegation](#input\_subnet\_delegation) | Subnet delegation enables you to designate a specific subnet for an Azure PaaS service | `string` | n/a | yes |
| <a name="input_subnet_name"></a> [subnet\_name](#input\_subnet\_name) | The subnet name. | `string` | n/a | yes |
| <a name="input_subscription_id"></a> [subscription\_id](#input\_subscription\_id) | The subscription id for the deployment. | `string` | n/a | yes |
| <a name="input_tags_map"></a> [tags\_map](#input\_tags\_map) | A map of tags to assign the same metadata to all resources in the environment. Format: key:value. | `map(string)` | <pre>{<br>  "creator": "tf",<br>  "env": "dev"<br>}</pre> | no |
| <a name="input_tiering_ssd_percent"></a> [tiering\_ssd\_percent](#input\_tiering\_ssd\_percent) | When set\_obs\_integration is true, this variable sets the capacity percentage of the filesystem that resides on SSD. For example, for an SSD with a total capacity of 20GB, and the tiering\_ssd\_percent is set to 20, the total available capacity is 100GB. | `number` | `20` | no |
| <a name="input_traces_per_ionode"></a> [traces\_per\_ionode](#input\_traces\_per\_ionode) | The number of traces per ionode. Traces are low-level events generated by Weka processes and are used as troubleshooting information for support purposes. | `number` | `10` | no |
| <a name="input_vm_username"></a> [vm\_username](#input\_vm\_username) | The user name for logging in to the virtual machines. | `string` | `"weka"` | no |
| <a name="input_vnet_name"></a> [vnet\_name](#input\_vnet\_name) | The virtual network name. | `string` | n/a | yes |
| <a name="input_vnet_rg_name"></a> [vnet\_rg\_name](#input\_vnet\_rg\_name) | Resource group name of vnet | `string` | n/a | yes |
| <a name="input_weka_version"></a> [weka\_version](#input\_weka\_version) | The Weka version to deploy. | `string` | `"4.2.0.142"` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_cluster_helpers_commands"></a> [cluster\_helpers\_commands](#output\_cluster\_helpers\_commands) | Useful commands and script to interact with weka cluster |
<!-- END_TF_DOCS -->
