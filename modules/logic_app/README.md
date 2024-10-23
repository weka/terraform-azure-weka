<!-- BEGIN_TF_DOCS -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | >= 1.4.6 |
| <a name="requirement_azurerm"></a> [azurerm](#requirement\_azurerm) | ~>4.6.0 |
| <a name="requirement_local"></a> [local](#requirement\_local) | ~>2.4.0 |
| <a name="requirement_null"></a> [null](#requirement\_null) | ~>3.2.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_azurerm"></a> [azurerm](#provider\_azurerm) | ~>4.6.0 |
| <a name="provider_local"></a> [local](#provider\_local) | ~>2.4.0 |
| <a name="provider_null"></a> [null](#provider\_null) | ~>3.2.0 |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [azurerm_key_vault_access_policy.standard_logic_app_get_secret_permission](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/key_vault_access_policy) | resource |
| [azurerm_logic_app_standard.logic_app_standard](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/logic_app_standard) | resource |
| [azurerm_service_plan.logicapp_service_plan](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/service_plan) | resource |
| [azurerm_storage_share.storage_share](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/storage_share) | resource |
| [azurerm_storage_share_directory.share_directory_scale_down](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/storage_share_directory) | resource |
| [azurerm_storage_share_directory.share_directory_scale_up](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/storage_share_directory) | resource |
| [azurerm_storage_share_file.connections_share_file](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/storage_share_file) | resource |
| [azurerm_storage_share_file.scale_down_share_file](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/storage_share_file) | resource |
| [azurerm_storage_share_file.scale_up_share_file](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/storage_share_file) | resource |
| [local_file.connections_workflow_file](https://registry.terraform.io/providers/hashicorp/local/latest/docs/resources/file) | resource |
| [local_file.scale_down_workflow_file](https://registry.terraform.io/providers/hashicorp/local/latest/docs/resources/file) | resource |
| [local_file.scale_up_workflow_file](https://registry.terraform.io/providers/hashicorp/local/latest/docs/resources/file) | resource |
| [null_resource.wait_for_logic_app](https://registry.terraform.io/providers/hashicorp/null/latest/docs/resources/resource) | resource |
| [azurerm_client_config.current](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/client_config) | data source |
| [azurerm_storage_account.logicapp](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/storage_account) | data source |
| [azurerm_storage_share.storage_share](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/storage_share) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_cluster_name"></a> [cluster\_name](#input\_cluster\_name) | Cluster name | `string` | n/a | yes |
| <a name="input_function_app_id"></a> [function\_app\_id](#input\_function\_app\_id) | The ID of the function app. | `string` | n/a | yes |
| <a name="input_function_app_key"></a> [function\_app\_key](#input\_function\_app\_key) | The key of the function app. | `string` | n/a | yes |
| <a name="input_function_app_name"></a> [function\_app\_name](#input\_function\_app\_name) | The name of the function app. | `string` | n/a | yes |
| <a name="input_key_vault_id"></a> [key\_vault\_id](#input\_key\_vault\_id) | The id of the Azure Key Vault. | `string` | n/a | yes |
| <a name="input_key_vault_uri"></a> [key\_vault\_uri](#input\_key\_vault\_uri) | The URI of the Azure Key Vault. | `string` | n/a | yes |
| <a name="input_location"></a> [location](#input\_location) | The Azure region to deploy all resources to. | `string` | n/a | yes |
| <a name="input_logic_app_identity_id"></a> [logic\_app\_identity\_id](#input\_logic\_app\_identity\_id) | The ID of the managed identity for the logic app | `string` | n/a | yes |
| <a name="input_logic_app_identity_principal"></a> [logic\_app\_identity\_principal](#input\_logic\_app\_identity\_principal) | The principal ID of the managed identity for the logic app | `string` | n/a | yes |
| <a name="input_logic_app_subnet_delegation_id"></a> [logic\_app\_subnet\_delegation\_id](#input\_logic\_app\_subnet\_delegation\_id) | Required to specify if subnet\_name were used to specify pre-defined subnets for weka. Logicapp subnet delegation requires an additional subnet, and in the case of pre-defined networking this one also should be pre-defined | `string` | n/a | yes |
| <a name="input_prefix"></a> [prefix](#input\_prefix) | Prefix for all resources | `string` | n/a | yes |
| <a name="input_restricted_inbound_access"></a> [restricted\_inbound\_access](#input\_restricted\_inbound\_access) | Restrict inbound access to internal VNet | `bool` | n/a | yes |
| <a name="input_rg_name"></a> [rg\_name](#input\_rg\_name) | A predefined resource group in the Azure subscription. | `string` | n/a | yes |
| <a name="input_storage_account_name"></a> [storage\_account\_name](#input\_storage\_account\_name) | The name of the storage account. | `string` | n/a | yes |
| <a name="input_subnet_id"></a> [subnet\_id](#input\_subnet\_id) | The ID of the cluster subnet. | `string` | n/a | yes |
| <a name="input_use_secured_storage_account"></a> [use\_secured\_storage\_account](#input\_use\_secured\_storage\_account) | Use secured storage account with logic app. | `bool` | `false` | no |

## Outputs

No outputs.
<!-- END_TF_DOCS -->
