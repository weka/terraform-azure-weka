<!-- BEGIN_TF_DOCS -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | >= 1.3.7 |
| <a name="requirement_azuread"></a> [azuread](#requirement\_azuread) | >= 2.33.0 |
| <a name="requirement_azurerm"></a> [azurerm](#requirement\_azurerm) | ~> 3.43.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_azuread"></a> [azuread](#provider\_azuread) | >= 2.33.0 |
| <a name="provider_azurerm"></a> [azurerm](#provider\_azurerm) | ~> 3.43.0 |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [azuread_application.main](https://registry.terraform.io/providers/hashicorp/azuread/latest/docs/resources/application) | resource |
| [azuread_directory_role.main](https://registry.terraform.io/providers/hashicorp/azuread/latest/docs/resources/directory_role) | resource |
| [azuread_directory_role_assignment.main](https://registry.terraform.io/providers/hashicorp/azuread/latest/docs/resources/directory_role_assignment) | resource |
| [azuread_service_principal.main](https://registry.terraform.io/providers/hashicorp/azuread/latest/docs/resources/service_principal) | resource |
| [azuread_service_principal_password.main](https://registry.terraform.io/providers/hashicorp/azuread/latest/docs/resources/service_principal_password) | resource |
| [azurerm_role_assignment.main](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment) | resource |
| [azuread_client_config.current](https://registry.terraform.io/providers/hashicorp/azuread/latest/docs/data-sources/client_config) | data source |
| [azurerm_subscription.primary](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/subscription) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_assignments"></a> [assignments](#input\_assignments) | The list of role assignments to this service principal | <pre>list(object({<br>    scope                = string<br>    role_definition_name = string<br>  }))</pre> | <pre>[<br>  {<br>    "role_definition_name": "Owner",<br>    "scope": "/subscriptions/d2f248b9-d054-477f-b7e8-413921532c2a"<br>  }<br>]</pre> | no |
| <a name="input_azure_role_description"></a> [azure\_role\_description](#input\_azure\_role\_description) | The description for this Role Assignment | `any` | `null` | no |
| <a name="input_azure_role_name"></a> [azure\_role\_name](#input\_azure\_role\_name) | A unique UUID/GUID for this Role Assignment - one will be generated if not specified. | `any` | `null` | no |
| <a name="input_description"></a> [description](#input\_description) | Description of the service principal | `string` | `"Github CI user"` | no |
| <a name="input_role_definition_name"></a> [role\_definition\_name](#input\_role\_definition\_name) | built-in role for the service principal | `any` | `null` | no |
| <a name="input_service_principal_name"></a> [service\_principal\_name](#input\_service\_principal\_name) | Name of the service principal | `string` | `"CIUser"` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_client_id"></a> [client\_id](#output\_client\_id) | The application id of AzureAD application created. |
| <a name="output_client_secret"></a> [client\_secret](#output\_client\_secret) | Password for service principal. |
| <a name="output_service_principal_name"></a> [service\_principal\_name](#output\_service\_principal\_name) | Service principal name |
| <a name="output_service_principal_object_id"></a> [service\_principal\_object\_id](#output\_service\_principal\_object\_id) | Object id of service principal |
<!-- END_TF_DOCS -->