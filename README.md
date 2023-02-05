# Azure-weka deployment Terraform package
The Weka cluster on Azure provides a fast and scalable platform to run, for example, performance-intensive applications and hybrid cloud workflows. It can also be used for object stores, tiering, and snapshots using the Azure Blob service.
The provided Azure-Weka Terraform package contains modules and examples you can customize according to your deployment needs. The installation is based on applying the customized Terraform variables file to a predefined Azure subscription. 
Applying the Terraform variables file performs the following:
- Creates resources in a predefined resource group, such as virtual machines, network interfaces, function app, load balancer, and more.
- Deploys Azure virtual machines.
- Installs the Weka software.
- Configures the Weka cluster.

## Usage
```hcl
module "deploy-weka" {
   source                = "../.."
   prefix                = "weka"
   rg_name               = "myResourceGroup"
   vnet_name             = "weka-vpc-0"
   subnets               = ["weka-subnet-0"]
   sg_id                 = "security-group-id"
   get_weka_io_token     = "get_weka_io_token"
   cluster_name          = "myCluster"
   set_obs_integration   = false
   instance_type         = "Standard_L8s_v3"
   cluster_size          = 6
   private_dns_zone_name = "myDns.private.net"
}
```

<!-- BEGIN_TF_DOCS -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_azurerm"></a> [azurerm](#requirement\_azurerm) | ~>3.26.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_archive"></a> [archive](#provider\_archive) | n/a |
| <a name="provider_azuread"></a> [azuread](#provider\_azuread) | n/a |
| <a name="provider_azurerm"></a> [azurerm](#provider\_azurerm) | ~>3.26.0 |
| <a name="provider_local"></a> [local](#provider\_local) | n/a |
| <a name="provider_null"></a> [null](#provider\_null) | n/a |
| <a name="provider_random"></a> [random](#provider\_random) | n/a |
| <a name="provider_template"></a> [template](#provider\_template) | n/a |
| <a name="provider_tls"></a> [tls](#provider\_tls) | n/a |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [azuread_application.function_app](https://registry.terraform.io/providers/hashicorp/azuread/latest/docs/resources/application) | resource |
| [azuread_service_principal.function-app-principal](https://registry.terraform.io/providers/hashicorp/azuread/latest/docs/resources/service_principal) | resource |
| [azurerm_application_insights.application_insights](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/application_insights) | resource |
| [azurerm_key_vault.key_vault](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/key_vault) | resource |
| [azurerm_key_vault_access_policy.function-app-get-secret-permission](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/key_vault_access_policy) | resource |
| [azurerm_key_vault_access_policy.key_vault_access_policy](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/key_vault_access_policy) | resource |
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
| [azurerm_private_dns_a_record.dns_a_record_backend_lb](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/private_dns_a_record) | resource |
| [azurerm_proximity_placement_group.ppg](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/proximity_placement_group) | resource |
| [azurerm_public_ip_prefix.public_ip_prefix](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/public_ip_prefix) | resource |
| [azurerm_role_assignment.function-app-key-vault-secrets-user](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_role_assignment.function-app-reader](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_role_assignment.function-app-scale-set-machine-owner](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_role_assignment.function-assignment](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_role_assignment.storage-account-contributor](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_role_assignment.storage-blob-data-owner](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_role_assignment.vm_role_assignment](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_service_plan.app_service_plan](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/service_plan) | resource |
| [azurerm_storage_account.deployment_sa](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/storage_account) | resource |
| [azurerm_storage_blob.function_app_code](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/storage_blob) | resource |
| [azurerm_storage_blob.state](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/storage_blob) | resource |
| [azurerm_storage_container.deployment](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/storage_container) | resource |
| [local_file.private_key](https://registry.terraform.io/providers/hashicorp/local/latest/docs/resources/file) | resource |
| [local_file.public_key](https://registry.terraform.io/providers/hashicorp/local/latest/docs/resources/file) | resource |
| [null_resource.build_function_code](https://registry.terraform.io/providers/hashicorp/null/latest/docs/resources/resource) | resource |
| [random_password.weka_password](https://registry.terraform.io/providers/hashicorp/random/latest/docs/resources/password) | resource |
| [tls_private_key.ssh_key](https://registry.terraform.io/providers/hashicorp/tls/latest/docs/resources/private_key) | resource |
| [archive_file.function_zip](https://registry.terraform.io/providers/hashicorp/archive/latest/docs/data-sources/file) | data source |
| [azuread_client_config.function-app-client-config](https://registry.terraform.io/providers/hashicorp/azuread/latest/docs/data-sources/client_config) | data source |
| [azurerm_client_config.current](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/client_config) | data source |
| [azurerm_function_app_host_keys.function_keys](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/function_app_host_keys) | data source |
| [azurerm_resource_group.rg](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/resource_group) | data source |
| [azurerm_subnet.subnets](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/subnet) | data source |
| [azurerm_subscription.primary](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/subscription) | data source |
| [azurerm_virtual_network.vnet](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/virtual_network) | data source |
| [template_cloudinit_config.cloud_init](https://registry.terraform.io/providers/hashicorp/template/latest/docs/data-sources/cloudinit_config) | data source |
| [template_file.init](https://registry.terraform.io/providers/hashicorp/template/latest/docs/data-sources/file) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_apt_repo_url"></a> [apt\_repo\_url](#input\_apt\_repo\_url) | The URL of the apt private repository. | `string` | `""` | no |
| <a name="input_blob_obs_access_key"></a> [blob\_obs\_access\_key](#input\_blob\_obs\_access\_key) | The access key of the existing Blob object store container. | `string` | `""` | no |
| <a name="input_cluster_name"></a> [cluster\_name](#input\_cluster\_name) | The cluster name. | `string` | `"poc"` | no |
| <a name="input_cluster_size"></a> [cluster\_size](#input\_cluster\_size) | The number of virtual machines to deploy. | `number` | `6` | no |
| <a name="input_container_number_map"></a> [container\_number\_map](#input\_container\_number\_map) | Maps the number of objects and memory size per machine type. | <pre>map(object({<br>    compute  = number<br>    drive    = number<br>    frontend = number<br>    nvme     = number<br>    nics     = number<br>    memory   = string<br>  }))</pre> | <pre>{<br>  "Standard_L16s_v3": {<br>    "compute": 4,<br>    "drive": 2,<br>    "frontend": 1,<br>    "memory": "72GB",<br>    "nics": 8,<br>    "nvme": 2<br>  },<br>  "Standard_L8s_v3": {<br>    "compute": 1,<br>    "drive": 1,<br>    "frontend": 1,<br>    "memory": "31GB",<br>    "nics": 4,<br>    "nvme": 1<br>  }<br>}</pre> | no |
| <a name="input_default_disk_size"></a> [default\_disk\_size](#input\_default\_disk\_size) | The default disk size. | `number` | `48` | no |
| <a name="input_get_weka_io_token"></a> [get\_weka\_io\_token](#input\_get\_weka\_io\_token) | The token to download the Weka release from get.weka.io. | `string` | `""` | no |
| <a name="input_hotspare"></a> [hotspare](#input\_hotspare) | Hot-spare value. | `number` | `1` | no |
| <a name="input_install_ofed_url"></a> [install\_ofed\_url](#input\_install\_ofed\_url) | The URL of the Blob with the OFED tgz file. | `string` | `""` | no |
| <a name="input_install_weka_url"></a> [install\_weka\_url](#input\_install\_weka\_url) | The URL of the Weka release download tar file. | `string` | `""` | no |
| <a name="input_instance_type"></a> [instance\_type](#input\_instance\_type) | The virtual machine type (sku) to deploy. | `string` | `"Standard_L8s_v3"` | no |
| <a name="input_linux_vm_image"></a> [linux\_vm\_image](#input\_linux\_vm\_image) | The source image reference. | `map(string)` | <pre>{<br>  "offer": "UbuntuServer",<br>  "publisher": "Canonical",<br>  "sku": "18.04-LTS",<br>  "version": "latest"<br>}</pre> | no |
| <a name="input_obs_container_name"></a> [obs\_container\_name](#input\_obs\_container\_name) | Name of existing obs conatiner name | `string` | `""` | no |
| <a name="input_obs_name"></a> [obs\_name](#input\_obs\_name) | Name of existing obs storage account | `string` | `""` | no |
| <a name="input_ofed_version"></a> [ofed\_version](#input\_ofed\_version) | The OFED driver version to for ubuntu 18. | `string` | `"5.7-1.0.2.0"` | no |
| <a name="input_prefix"></a> [prefix](#input\_prefix) | The prefix for all the resource names. For example, the prefix for your system name. | `string` | `"weka"` | no |
| <a name="input_private_dns_zone_name"></a> [private\_dns\_zone\_name](#input\_private\_dns\_zone\_name) | The private DNS zone name. | `string` | n/a | yes |
| <a name="input_private_network"></a> [private\_network](#input\_private\_network) | Determines whether to enable a private or public network. The default is public network. | `bool` | `false` | no |
| <a name="input_protection_level"></a> [protection\_level](#input\_protection\_level) | Cluster data protection level. | `number` | `2` | no |
| <a name="input_rg_name"></a> [rg\_name](#input\_rg\_name) | A predefined resource group in the Azure subscription. | `string` | n/a | yes |
| <a name="input_set_obs_integration"></a> [set\_obs\_integration](#input\_set\_obs\_integration) | Determines whether to enable object stores integration with the Weka cluster. Set true to enable the integration. | `bool` | `false` | no |
| <a name="input_sg_id"></a> [sg\_id](#input\_sg\_id) | The security group id. | `string` | n/a | yes |
| <a name="input_ssh_private_key"></a> [ssh\_private\_key](#input\_ssh\_private\_key) | The path to the VM private key. If it is not set, the key is auto-generated. If it is set, also set the ssh\_private\_key. The private key used for connecting to the deployed virtual machines to initiate the clusterization of Weka. | `string` | `null` | no |
| <a name="input_ssh_public_key"></a> [ssh\_public\_key](#input\_ssh\_public\_key) | The path to the VM public key. If it is not set, the key is auto-generated. If it is set, also set the ssh\_private\_key. | `string` | `null` | no |
| <a name="input_stripe_width"></a> [stripe\_width](#input\_stripe\_width) | Stripe width = cluster\_size - protection\_level - 1 (by default). | `number` | `-1` | no |
| <a name="input_subnets"></a> [subnets](#input\_subnets) | The subnet names list. | `list(string)` | n/a | yes |
| <a name="input_subscription_id"></a> [subscription\_id](#input\_subscription\_id) | The subscription id for the deployment. | `string` | n/a | yes |
| <a name="input_tags_map"></a> [tags\_map](#input\_tags\_map) | A map of tags to assign the same metadata to all resources in the environment. Format: key:value. | `map(string)` | <pre>{<br>  "creator": "tf",<br>  "env": "dev"<br>}</pre> | no |
| <a name="input_tiering_ssd_percent"></a> [tiering\_ssd\_percent](#input\_tiering\_ssd\_percent) | When set\_obs\_integration is true, this variable sets the capacity percentage of the filesystem that resides on SSD. For example, for an SSD with a total capacity of 20GB, and the tiering\_ssd\_percent is set to 20, the total available capacity is 100GB. | `number` | `20` | no |
| <a name="input_traces_per_ionode"></a> [traces\_per\_ionode](#input\_traces\_per\_ionode) | The number of traces per ionode. Traces are low-level events generated by Weka processes and are used as troubleshooting information for support purposes. | `number` | `10` | no |
| <a name="input_vm_username"></a> [vm\_username](#input\_vm\_username) | The user name for logging in to the virtual machines. | `string` | `"weka"` | no |
| <a name="input_vnet_name"></a> [vnet\_name](#input\_vnet\_name) | The virtual network name. | `string` | n/a | yes |
| <a name="input_weka_version"></a> [weka\_version](#input\_weka\_version) | The Weka version to deploy. | `string` | `"4.1.0.71"` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_get-cluster-status"></a> [get-cluster-status](#output\_get-cluster-status) | A command to get the cluster status. |
| <a name="output_get-vms-private-ips"></a> [get-vms-private-ips](#output\_get-vms-private-ips) | Get private ips of vms |
| <a name="output_get-vms-public-ips"></a> [get-vms-public-ips](#output\_get-vms-public-ips) | Get public ips of vms |
| <a name="output_ssh-key-download-blob"></a> [ssh-key-download-blob](#output\_ssh-key-download-blob) | Commands to download the ssh keys from the Azure Blob. |
| <a name="output_ssh-key-files-path"></a> [ssh-key-files-path](#output\_ssh-key-files-path) | Displays the path of the ssh keys. |
<!-- END_TF_DOCS -->
