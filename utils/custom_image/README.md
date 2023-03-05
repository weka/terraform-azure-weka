## Create custom weka image with ofed
- run packer for create image

  change the variables at Taskfile.yaml

  RUN **task create-azure-weka-image**

- Deploy azure gallery for shared vm:

  change the variables at vars.auto.tfvars file

  change or remove the backend.tf file

  RUN **task create-shared-gallery**


<!-- BEGIN_TF_DOCS -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | >= 1.3.7 |
| <a name="requirement_azurerm"></a> [azurerm](#requirement\_azurerm) | ~>3.44.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_azurerm"></a> [azurerm](#provider\_azurerm) | ~>3.44.0 |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [azurerm_resource_group.rg](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/resource_group) | resource |
| [azurerm_shared_image.shared_image](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/shared_image) | resource |
| [azurerm_shared_image_gallery.shared_image_gallery](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/shared_image_gallery) | resource |
| [azurerm_shared_image_version.shared_image_version](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/shared_image_version) | resource |
| [azurerm_images.image](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/images) | data source |
| [azurerm_resource_group.get-rg](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/resource_group) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_custom_image_name"></a> [custom\_image\_name](#input\_custom\_image\_name) | Name of created custom image | `string` | `"weka-custom-image"` | no |
| <a name="input_custom_vm_version"></a> [custom\_vm\_version](#input\_custom\_vm\_version) | n/a | `string` | `"1.0.0"` | no |
| <a name="input_location"></a> [location](#input\_location) | n/a | `string` | `"East US"` | no |
| <a name="input_ofed_version"></a> [ofed\_version](#input\_ofed\_version) | The OFED driver version to for ubuntu 18. | `string` | `"5.7-1.0.2.0"` | no |
| <a name="input_prefix"></a> [prefix](#input\_prefix) | n/a | `string` | `"weka-image"` | no |
| <a name="input_rg_name"></a> [rg\_name](#input\_rg\_name) | n/a | `string` | n/a | yes |
| <a name="input_subscription_id"></a> [subscription\_id](#input\_subscription\_id) | n/a | `string` | n/a | yes |
| <a name="input_tags"></a> [tags](#input\_tags) | n/a | `map(string)` | <pre>{<br>  "creator": "tf"<br>}</pre> | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_custom-vm-id"></a> [custom-vm-id](#output\_custom-vm-id) | n/a |
| <a name="output_gallery-name"></a> [gallery-name](#output\_gallery-name) | n/a |
| <a name="output_image-identifier"></a> [image-identifier](#output\_image-identifier) | n/a |
| <a name="output_image-name"></a> [image-name](#output\_image-name) | n/a |
| <a name="output_managed_image_id"></a> [managed\_image\_id](#output\_managed\_image\_id) | n/a |
<!-- END_TF_DOCS -->