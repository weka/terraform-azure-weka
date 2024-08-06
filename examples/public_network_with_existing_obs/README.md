<!-- BEGIN_TF_DOCS -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | >= 1.3.7 |
| <a name="requirement_azurerm"></a> [azurerm](#requirement\_azurerm) | ~> 3.114.0 |

## Providers

No providers.

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_weka_deployment"></a> [weka\_deployment](#module\_weka\_deployment) | ../.. | n/a |

## Resources

No resources.

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_get_weka_io_token"></a> [get\_weka\_io\_token](#input\_get\_weka\_io\_token) | Get get.weka.io token for downloading weka | `string` | n/a | yes |
| <a name="input_subscription_id"></a> [subscription\_id](#input\_subscription\_id) | Subscription id for deployment | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_weka_deployment_output"></a> [weka\_deployment\_output](#output\_weka\_deployment\_output) | n/a |
<!-- END_TF_DOCS -->
