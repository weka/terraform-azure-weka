<!-- BEGIN_TF_DOCS -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_azurerm"></a> [azurerm](#requirement\_azurerm) | ~>3.50.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_azurerm"></a> [azurerm](#provider\_azurerm) | ~>3.50.0 |

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_create-sa"></a> [create-sa](#module\_create-sa) | ./create_sa | n/a |

## Resources

| Name | Type |
|------|------|
| [azurerm_resource_group.rg](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/resource_group) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_resource_group_name"></a> [resource\_group\_name](#input\_resource\_group\_name) | The name of Azure resource group | `string` | `"weka-tf-functions"` | no |
| <a name="input_subscription_id"></a> [subscription\_id](#input\_subscription\_id) | Subscription id for deployment | `string` | n/a | yes |
| <a name="input_supported_regions_file_path"></a> [supported\_regions\_file\_path](#input\_supported\_regions\_file\_path) | Path to supported regions file | `string` | `"supported_regions/release.txt"` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_supported_regions"></a> [supported\_regions](#output\_supported\_regions) | Supported regions list (for release) |
<!-- END_TF_DOCS -->
