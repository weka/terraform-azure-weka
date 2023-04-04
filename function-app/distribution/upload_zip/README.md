<!-- BEGIN_TF_DOCS -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_azurerm"></a> [azurerm](#requirement\_azurerm) | ~>3.50.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_azurerm"></a> [azurerm](#provider\_azurerm) | ~>3.50.0 |
| <a name="provider_null"></a> [null](#provider\_null) | n/a |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [azurerm_storage_account.sa](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/storage_account) | resource |
| [azurerm_storage_container.container](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/storage_container) | resource |
| [azurerm_storage_management_policy.retention_policy](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/storage_management_policy) | resource |
| [null_resource.function_app_code](https://registry.terraform.io/providers/hashicorp/null/latest/docs/resources/resource) | resource |
| [azurerm_storage_account_blob_container_sas.sa_sas](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/storage_account_blob_container_sas) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_dist"></a> [dist](#input\_dist) | Distribution option ('dev' or 'release') | `string` | `"dev"` | no |
| <a name="input_function_app_code_hash"></a> [function\_app\_code\_hash](#input\_function\_app\_code\_hash) | The MD5 checksum of function app dir. | `string` | n/a | yes |
| <a name="input_function_app_zip_path"></a> [function\_app\_zip\_path](#input\_function\_app\_zip\_path) | An absolute path to the code zip on the local system. | `string` | n/a | yes |
| <a name="input_region"></a> [region](#input\_region) | Azure location (region) | `string` | n/a | yes |
| <a name="input_rg_name"></a> [rg\_name](#input\_rg\_name) | Resource group name. | `string` | n/a | yes |

## Outputs

No outputs.
<!-- END_TF_DOCS -->