<!-- BEGIN_TF_DOCS -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | >= 1.3.7 |
| <a name="requirement_azurerm"></a> [azurerm](#requirement\_azurerm) | ~> 3.43.0 |

## Providers

No providers.

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_create-network"></a> [create-network](#module\_create-network) | ../../modules/create_networks | n/a |
| <a name="module_deploy-weka"></a> [deploy-weka](#module\_deploy-weka) | ../.. | n/a |

## Resources

No resources.

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_address_space"></a> [address\_space](#input\_address\_space) | address space that is used the virtual network. | `string` | n/a | yes |
| <a name="input_cluster_name"></a> [cluster\_name](#input\_cluster\_name) | Cluster name | `string` | n/a | yes |
| <a name="input_cluster_size"></a> [cluster\_size](#input\_cluster\_size) | Weka cluster size | `number` | n/a | yes |
| <a name="input_custom_image_id"></a> [custom\_image\_id](#input\_custom\_image\_id) | Custom image id; /subscriptions/<subscription id>/resourceGroups/<rg name>/providers/Microsoft.Compute/images/<image name> | `string` | n/a | yes |
| <a name="input_get_weka_io_token"></a> [get\_weka\_io\_token](#input\_get\_weka\_io\_token) | Get get.weka.io token for downloading weka | `string` | n/a | yes |
| <a name="input_install_ofed"></a> [install\_ofed](#input\_install\_ofed) | Install ofed for weka cluster with dpdk configuration | `bool` | n/a | yes |
| <a name="input_instance_type"></a> [instance\_type](#input\_instance\_type) | The SKU which should be used for this virtual machine | `string` | n/a | yes |
| <a name="input_prefix"></a> [prefix](#input\_prefix) | Prefix for all resources | `string` | n/a | yes |
| <a name="input_rg_name"></a> [rg\_name](#input\_rg\_name) | Name of existing resource group | `string` | n/a | yes |
| <a name="input_set_obs_integration"></a> [set\_obs\_integration](#input\_set\_obs\_integration) | Should be true to enable OBS integration with weka cluster | `bool` | n/a | yes |
| <a name="input_subnet_delegation"></a> [subnet\_delegation](#input\_subnet\_delegation) | Subnet delegation enables you to designate a specific subnet for an Azure PaaS service | `string` | n/a | yes |
| <a name="input_subnet_prefixes"></a> [subnet\_prefixes](#input\_subnet\_prefixes) | List of address prefixes to use for the subnet | `list(string)` | n/a | yes |
| <a name="input_subscription_id"></a> [subscription\_id](#input\_subscription\_id) | Subscription id for deployment | `string` | n/a | yes |
| <a name="input_tiering_ssd_percent"></a> [tiering\_ssd\_percent](#input\_tiering\_ssd\_percent) | When OBS integration set to true , this parameter sets how much of the filesystem capacity should reside on SSD. For example, if this parameter is 20 and the total available SSD capacity is 20GB, the total capacity would be 100GB | `number` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_get-cluster-helpers-commands"></a> [get-cluster-helpers-commands](#output\_get-cluster-helpers-commands) | n/a |
<!-- END_TF_DOCS -->