<!-- BEGIN_TF_DOCS -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_azurerm"></a> [azurerm](#requirement\_azurerm) | ~>3.43.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_archive"></a> [archive](#provider\_archive) | n/a |
| <a name="provider_null"></a> [null](#provider\_null) | n/a |

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_upload-zip"></a> [upload-zip](#module\_upload-zip) | ./upload_zip | n/a |

## Resources

| Name | Type |
|------|------|
| [null_resource.build_function_code](https://registry.terraform.io/providers/hashicorp/null/latest/docs/resources/resource) | resource |
| [archive_file.function_zip](https://registry.terraform.io/providers/hashicorp/archive/latest/docs/data-sources/file) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_dist"></a> [dist](#input\_dist) | Distribution option ('dev' or 'release') | `string` | `"dev"` | no |
| <a name="input_regions"></a> [regions](#input\_regions) | Map of lists of supported regions | `map(list(string))` | <pre>{<br>  "dev": [<br>    "eastus"<br>  ],<br>  "release": [<br>    "brazilsouth",<br>    "canadacentral",<br>    "centralus",<br>    "eastus",<br>    "eastus2",<br>    "southcentralus",<br>    "westus2",<br>    "westus3",<br>    "francecentral",<br>    "germanywestcentral",<br>    "northeurope",<br>    "westeurope",<br>    "uksouth",<br>    "swedencentral",<br>    "qatarcentral",<br>    "australiaeast",<br>    "centralindia",<br>    "japaneast",<br>    "eastasia",<br>    "southeastasia"<br>  ]<br>}</pre> | no |
| <a name="input_subscription_id"></a> [subscription\_id](#input\_subscription\_id) | Subscription id for deployment | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_function_app_zip_filepath"></a> [function\_app\_zip\_filepath](#output\_function\_app\_zip\_filepath) | Function app code zip path |
| <a name="output_function_app_zip_md5"></a> [function\_app\_zip\_md5](#output\_function\_app\_zip\_md5) | Function app code dir MD5 |
<!-- END_TF_DOCS -->