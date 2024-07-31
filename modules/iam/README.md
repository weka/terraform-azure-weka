<!-- BEGIN_TF_DOCS -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | >= 1.4.6 |
| <a name="requirement_azurerm"></a> [azurerm](#requirement\_azurerm) | ~>3.75.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_azurerm"></a> [azurerm](#provider\_azurerm) | ~>3.75.0 |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [azurerm_role_assignment.function_app_key_vault_secrets_user](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_role_assignment.function_app_reader](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_role_assignment.function_app_scale_set_machine_owner](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_role_assignment.key_vault_set_secret](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_role_assignment.logic_app_standard_reader](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_role_assignment.logic_app_standard_reader_secret](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_role_assignment.logic_app_standard_reader_smb_data](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_role_assignment.managed_identity_operator](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_role_assignment.network_contributor](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_role_assignment.nfs_storage_blob_data_contributor](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_role_assignment.obs_data_contributor](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_role_assignment.obs_storage_blob_data_contributor](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_role_assignment.reader](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_role_assignment.storage_account_contributor](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_role_assignment.storage_blob_data_contributor](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_role_assignment.weka_tar_data_reader](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azurerm_role_definition.key_vault_set_secret](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_definition) | resource |
| [azurerm_user_assigned_identity.function_app](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/user_assigned_identity) | resource |
| [azurerm_user_assigned_identity.logic_app](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/user_assigned_identity) | resource |
| [azurerm_user_assigned_identity.vmss](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/user_assigned_identity) | resource |
| [azurerm_resource_group.rg](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/resource_group) | data source |
| [azurerm_storage_account.obs_sa](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/storage_account) | data source |
| [azurerm_user_assigned_identity.function_app](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/user_assigned_identity) | data source |
| [azurerm_user_assigned_identity.logic_app](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/user_assigned_identity) | data source |
| [azurerm_user_assigned_identity.vmss](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/user_assigned_identity) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_cluster_name"></a> [cluster\_name](#input\_cluster\_name) | Cluster name | `string` | n/a | yes |
| <a name="input_deployment_container_name"></a> [deployment\_container\_name](#input\_deployment\_container\_name) | The name of the container for the deployment. | `string` | n/a | yes |
| <a name="input_deployment_storage_account_id"></a> [deployment\_storage\_account\_id](#input\_deployment\_storage\_account\_id) | The id of the storage account for the deployment. | `string` | n/a | yes |
| <a name="input_function_app_identity_name"></a> [function\_app\_identity\_name](#input\_function\_app\_identity\_name) | The user assigned identity name for the function app (if empty - new one is created). | `string` | `""` | no |
| <a name="input_key_vault_id"></a> [key\_vault\_id](#input\_key\_vault\_id) | The id of the Azure Key Vault. | `string` | n/a | yes |
| <a name="input_logic_app_identity_name"></a> [logic\_app\_identity\_name](#input\_logic\_app\_identity\_name) | The user assigned identity name for the logic app (if empty - new one is created). | `string` | `""` | no |
| <a name="input_logic_app_storage_account_id"></a> [logic\_app\_storage\_account\_id](#input\_logic\_app\_storage\_account\_id) | The id of the storage account for the logic app. | `string` | n/a | yes |
| <a name="input_nfs_deployment_container_name"></a> [nfs\_deployment\_container\_name](#input\_nfs\_deployment\_container\_name) | The name of the container for the NFS deployment. | `string` | n/a | yes |
| <a name="input_nfs_protocol_gateways_number"></a> [nfs\_protocol\_gateways\_number](#input\_nfs\_protocol\_gateways\_number) | The number of NFS protocol gateways. | `number` | n/a | yes |
| <a name="input_obs_container_name"></a> [obs\_container\_name](#input\_obs\_container\_name) | The name of the container for the OBS. | `string` | n/a | yes |
| <a name="input_prefix"></a> [prefix](#input\_prefix) | Prefix for all resources | `string` | n/a | yes |
| <a name="input_rg_name"></a> [rg\_name](#input\_rg\_name) | A predefined resource group in the Azure subscription. | `string` | n/a | yes |
| <a name="input_support_logic_app"></a> [support\_logic\_app](#input\_support\_logic\_app) | Enable support for logic app. | `bool` | `true` | no |
| <a name="input_tiering_enable_obs_integration"></a> [tiering\_enable\_obs\_integration](#input\_tiering\_enable\_obs\_integration) | Enable OBS integration for tiering. | `bool` | n/a | yes |
| <a name="input_tiering_obs_name"></a> [tiering\_obs\_name](#input\_tiering\_obs\_name) | Name of existing obs storage account. | `string` | `""` | no |
| <a name="input_vmss_identity_name"></a> [vmss\_identity\_name](#input\_vmss\_identity\_name) | The user assigned identity name for the vmss instances (if empty - new one is created). | `string` | `""` | no |
| <a name="input_weka_tar_storage_account_id"></a> [weka\_tar\_storage\_account\_id](#input\_weka\_tar\_storage\_account\_id) | n/a | `string` | `""` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_function_app_identity_client_id"></a> [function\_app\_identity\_client\_id](#output\_function\_app\_identity\_client\_id) | The client ID of the managed identity for the function app |
| <a name="output_function_app_identity_id"></a> [function\_app\_identity\_id](#output\_function\_app\_identity\_id) | The ID of the managed identity for the function app |
| <a name="output_function_app_identity_principal_id"></a> [function\_app\_identity\_principal\_id](#output\_function\_app\_identity\_principal\_id) | The principal ID of the managed identity for the function app |
| <a name="output_logic_app_identity_id"></a> [logic\_app\_identity\_id](#output\_logic\_app\_identity\_id) | The ID of the managed identity for the logic app |
| <a name="output_logic_app_identity_principal_id"></a> [logic\_app\_identity\_principal\_id](#output\_logic\_app\_identity\_principal\_id) | The principal ID of the managed identity for the logic app |
| <a name="output_vmss_identity_id"></a> [vmss\_identity\_id](#output\_vmss\_identity\_id) | The ID of the managed identity for the vmss |
<!-- END_TF_DOCS -->
