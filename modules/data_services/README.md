<!-- BEGIN_TF_DOCS -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | >= 1.4.6 |
| <a name="requirement_azurerm"></a> [azurerm](#requirement\_azurerm) | ~>4.6.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_azurerm"></a> [azurerm](#provider\_azurerm) | ~>4.6.0 |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [azurerm_key_vault_access_policy.data_services_key_vault](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/key_vault_access_policy) | resource |
| [azurerm_linux_virtual_machine_scale_set.this](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/linux_virtual_machine_scale_set) | resource |
| [azurerm_role_assignment.data_services_key_vault](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_role_assignment.reader](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_role_assignment.weka_tar_data_reader](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_user_assigned_identity.this](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/user_assigned_identity) | resource |
| [azurerm_client_config.current](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/client_config) | data source |
| [azurerm_resource_group.rg](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/resource_group) | data source |
| [azurerm_subnet.subnet](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/subnet) | data source |
| [azurerm_user_assigned_identity.this](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/user_assigned_identity) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_apt_repo_server"></a> [apt\_repo\_server](#input\_apt\_repo\_server) | The URL of the apt private repository. | `string` | `""` | no |
| <a name="input_assign_public_ip"></a> [assign\_public\_ip](#input\_assign\_public\_ip) | Determines whether to assign public ip. | `bool` | `true` | no |
| <a name="input_cluster_name"></a> [cluster\_name](#input\_cluster\_name) | The cluster name. | `string` | n/a | yes |
| <a name="input_data_services_name"></a> [data\_services\_name](#input\_data\_services\_name) | The data services group name. | `string` | n/a | yes |
| <a name="input_data_services_number"></a> [data\_services\_number](#input\_data\_services\_number) | The number of virtual machines to deploy as data services. | `number` | n/a | yes |
| <a name="input_deploy_function_url"></a> [deploy\_function\_url](#input\_deploy\_function\_url) | The URL of deploy function from function app. | `string` | n/a | yes |
| <a name="input_disk_size"></a> [disk\_size](#input\_disk\_size) | The data services' weka disk size. | `number` | n/a | yes |
| <a name="input_function_app_default_key"></a> [function\_app\_default\_key](#input\_function\_app\_default\_key) | The default key of the function app. | `string` | n/a | yes |
| <a name="input_image_offer"></a> [image\_offer](#input\_image\_offer) | Azure Marketplace image offer (e.g., '0001-com-ubuntu-server-jammy' for Ubuntu 22.04 LTS). Required when image\_sku is set. Common offers: 0001-com-ubuntu-server-focal (20.04), 0001-com-ubuntu-server-jammy (22.04). | `string` | `null` | no |
| <a name="input_image_sku"></a> [image\_sku](#input\_image\_sku) | Azure Marketplace image SKU (e.g., '22\_04-lts-gen2' for Ubuntu 22.04 LTS). Required when using marketplace images. Must be paired with image\_offer. Use underscores, not dots. When set, source\_image\_id is ignored. | `string` | `null` | no |
| <a name="input_image_version"></a> [image\_version](#input\_image\_version) | Azure Marketplace image version (e.g., 'latest' or a specific version). Only used when image\_sku is set. Use 'latest' for the most recent version, or pin to a specific version for reproducibility. | `string` | `"latest"` | no |
| <a name="input_instance_type"></a> [instance\_type](#input\_instance\_type) | The virtual machine type (sku) to deploy. | `string` | n/a | yes |
| <a name="input_key_vault_id"></a> [key\_vault\_id](#input\_key\_vault\_id) | The id of the Azure Key Vault. | `string` | n/a | yes |
| <a name="input_location"></a> [location](#input\_location) | The Azure region to deploy all resources to. | `string` | n/a | yes |
| <a name="input_report_function_url"></a> [report\_function\_url](#input\_report\_function\_url) | The URL of report function from function app. | `string` | n/a | yes |
| <a name="input_rg_name"></a> [rg\_name](#input\_rg\_name) | A predefined resource group in the Azure subscription. | `string` | n/a | yes |
| <a name="input_root_volume_size"></a> [root\_volume\_size](#input\_root\_volume\_size) | The data services' root disk size. | `number` | `null` | no |
| <a name="input_sg_id"></a> [sg\_id](#input\_sg\_id) | Security group id. | `string` | n/a | yes |
| <a name="input_source_image_id"></a> [source\_image\_id](#input\_source\_image\_id) | Use weka custom image, ubuntu 20.04 with kernel 6.8.0. Ignored if image\_sku is set. | `string` | n/a | yes |
| <a name="input_ssh_public_key"></a> [ssh\_public\_key](#input\_ssh\_public\_key) | The VM public key. If it is not set, the keys are auto-generated. | `string` | n/a | yes |
| <a name="input_subnet_name"></a> [subnet\_name](#input\_subnet\_name) | The subnet names. | `string` | n/a | yes |
| <a name="input_tags_map"></a> [tags\_map](#input\_tags\_map) | A map of tags to assign the same metadata to all resources in the environment. Format: key:value. | `map(string)` | `{}` | no |
| <a name="input_vm_identity_name"></a> [vm\_identity\_name](#input\_vm\_identity\_name) | The name of the user assigned identity for the data services VMs. | `string` | `""` | no |
| <a name="input_vm_username"></a> [vm\_username](#input\_vm\_username) | The user name for logging in to the virtual machines. | `string` | `"weka"` | no |
| <a name="input_vnet_name"></a> [vnet\_name](#input\_vnet\_name) | The virtual network name. | `string` | n/a | yes |
| <a name="input_vnet_rg_name"></a> [vnet\_rg\_name](#input\_vnet\_rg\_name) | Resource group name of vnet | `string` | n/a | yes |
| <a name="input_weka_tar_storage_account_id"></a> [weka\_tar\_storage\_account\_id](#input\_weka\_tar\_storage\_account\_id) | n/a | `string` | `""` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_data_services_name"></a> [data\_services\_name](#output\_data\_services\_name) | Data services name |
| <a name="output_data_services_vmss_name"></a> [data\_services\_vmss\_name](#output\_data\_services\_vmss\_name) | The name of the data services VMSS. |
<!-- END_TF_DOCS -->
