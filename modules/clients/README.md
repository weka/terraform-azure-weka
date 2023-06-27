<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

| Name | Version |
|------|---------|
| <a name="provider_azurerm"></a> [azurerm](#provider\_azurerm) | n/a |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [azurerm_linux_virtual_machine.vms](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/linux_virtual_machine) | resource |
| [azurerm_network_interface.client_nic](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/network_interface) | resource |
| [azurerm_network_interface.primary_client_nic_private](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/network_interface) | resource |
| [azurerm_network_interface.primary_client_nic_public](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/network_interface) | resource |
| [azurerm_public_ip.public_ip](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/public_ip) | resource |
| [azurerm_resource_group.rg](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/resource_group) | data source |
| [azurerm_subnet.subnet](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/subnet) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_apt_repo_url"></a> [apt\_repo\_url](#input\_apt\_repo\_url) | The URL of the apt private repository. | `string` | `""` | no |
| <a name="input_assign_public_ip"></a> [assign\_public\_ip](#input\_assign\_public\_ip) | Determines whether to assign public ip. | `bool` | `true` | no |
| <a name="input_backend_ips"></a> [backend\_ips](#input\_backend\_ips) | n/a | `list(string)` | n/a | yes |
| <a name="input_clients_name"></a> [clients\_name](#input\_clients\_name) | The clients name. | `string` | n/a | yes |
| <a name="input_clients_number"></a> [clients\_number](#input\_clients\_number) | The number of virtual machines to deploy. | `number` | `2` | no |
| <a name="input_instance_type"></a> [instance\_type](#input\_instance\_type) | The virtual machine type (sku) to deploy. | `string` | n/a | yes |
| <a name="input_mount_clients_dpdk"></a> [mount\_clients\_dpdk](#input\_mount\_clients\_dpdk) | Install weka cluster with DPDK | `bool` | `true` | no |
| <a name="input_nics"></a> [nics](#input\_nics) | Number of nics to set on each client vm | `number` | `2` | no |
| <a name="input_nics_map"></a> [nics\_map](#input\_nics\_map) | n/a | `map(number)` | <pre>{<br>  "Standard_L16s_v3": 8,<br>  "Standard_L8s_v3": 4<br>}</pre> | no |
| <a name="input_ppg_id"></a> [ppg\_id](#input\_ppg\_id) | Placement proximity group id. | `string` | n/a | yes |
| <a name="input_preparation_template_file"></a> [preparation\_template\_file](#input\_preparation\_template\_file) | preparation\_template file path | `string` | n/a | yes |
| <a name="input_rg_name"></a> [rg\_name](#input\_rg\_name) | A predefined resource group in the Azure subscription. | `string` | n/a | yes |
| <a name="input_source_image_id"></a> [source\_image\_id](#input\_source\_image\_id) | Use weka custom image, ubuntu 20.04 with kernel 5.4 and ofed 5.8-1.1.2.1 | `string` | n/a | yes |
| <a name="input_ssh_public_key"></a> [ssh\_public\_key](#input\_ssh\_public\_key) | The VM public key. If it is not set, the keys are auto-generated. | `string` | n/a | yes |
| <a name="input_subnet_name"></a> [subnet\_name](#input\_subnet\_name) | The subnet names. | `string` | n/a | yes |
| <a name="input_vm_username"></a> [vm\_username](#input\_vm\_username) | The user name for logging in to the virtual machines. | `string` | `"weka"` | no |
| <a name="input_vnet_name"></a> [vnet\_name](#input\_vnet\_name) | The virtual network name. | `string` | n/a | yes |
| <a name="input_vnet_rg_name"></a> [vnet\_rg\_name](#input\_vnet\_rg\_name) | Resource group name of vnet | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_client-ips"></a> [client-ips](#output\_client-ips) | n/a |
<!-- END_TF_DOCS -->
