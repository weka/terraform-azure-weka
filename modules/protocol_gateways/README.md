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
| [azurerm_linux_virtual_machine_scale_set.vmss](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/linux_virtual_machine_scale_set) | resource |
| [azurerm_client_config.current](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/client_config) | data source |
| [azurerm_resource_group.rg](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/resource_group) | data source |
| [azurerm_subnet.subnet](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/subnet) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_apt_repo_server"></a> [apt\_repo\_server](#input\_apt\_repo\_server) | The URL of the apt private repository. | `string` | `""` | no |
| <a name="input_assign_public_ip"></a> [assign\_public\_ip](#input\_assign\_public\_ip) | Determines whether to assign public ip. | `bool` | `true` | no |
| <a name="input_backend_lb_ip"></a> [backend\_lb\_ip](#input\_backend\_lb\_ip) | The backend load balancer ip address. | `string` | n/a | yes |
| <a name="input_client_group_name"></a> [client\_group\_name](#input\_client\_group\_name) | Client access group name. | `string` | `"weka-cg"` | no |
| <a name="input_disk_size"></a> [disk\_size](#input\_disk\_size) | The disk size. | `number` | n/a | yes |
| <a name="input_frontend_num"></a> [frontend\_num](#input\_frontend\_num) | The number of frontend ionodes per instance. | `number` | `1` | no |
| <a name="input_gateways_name"></a> [gateways\_name](#input\_gateways\_name) | The protocol group name. | `string` | n/a | yes |
| <a name="input_gateways_number"></a> [gateways\_number](#input\_gateways\_number) | The number of virtual machines to deploy as protocol gateways. | `number` | n/a | yes |
| <a name="input_install_weka_url"></a> [install\_weka\_url](#input\_install\_weka\_url) | The URL of the Weka release download tar file. | `string` | n/a | yes |
| <a name="input_instance_type"></a> [instance\_type](#input\_instance\_type) | The virtual machine type (sku) to deploy. | `string` | n/a | yes |
| <a name="input_interface_group_name"></a> [interface\_group\_name](#input\_interface\_group\_name) | Interface group name. | `string` | `"weka-ig"` | no |
| <a name="input_key_vault_url"></a> [key\_vault\_url](#input\_key\_vault\_url) | The URL of the Azure Key Vault. | `string` | n/a | yes |
| <a name="input_nics_numbers"></a> [nics\_numbers](#input\_nics\_numbers) | Number of nics to set on each vm | `number` | `2` | no |
| <a name="input_ppg_id"></a> [ppg\_id](#input\_ppg\_id) | Placement proximity group id. | `string` | n/a | yes |
| <a name="input_protocol"></a> [protocol](#input\_protocol) | Name of the protocol. | `string` | `"NFS"` | no |
| <a name="input_rg_name"></a> [rg\_name](#input\_rg\_name) | A predefined resource group in the Azure subscription. | `string` | n/a | yes |
| <a name="input_secondary_ips_per_nic"></a> [secondary\_ips\_per\_nic](#input\_secondary\_ips\_per\_nic) | Number of secondary IPs per single NIC per protocol gateway virtual machine. | `number` | `3` | no |
| <a name="input_sg_id"></a> [sg\_id](#input\_sg\_id) | Security group id. | `string` | n/a | yes |
| <a name="input_smb_cluster_name"></a> [smb\_cluster\_name](#input\_smb\_cluster\_name) | The name of the SMB setup. | `string` | n/a | yes |
| <a name="input_smb_dns_ip_address"></a> [smb\_dns\_ip\_address](#input\_smb\_dns\_ip\_address) | DNS IP address | `string` | `""` | no |
| <a name="input_smb_domain_name"></a> [smb\_domain\_name](#input\_smb\_domain\_name) | The domain to join the SMB cluster to. | `string` | n/a | yes |
| <a name="input_smb_domain_netbios_name"></a> [smb\_domain\_netbios\_name](#input\_smb\_domain\_netbios\_name) | The domain NetBIOS name of the SMB cluster. | `string` | `""` | no |
| <a name="input_smb_domain_password"></a> [smb\_domain\_password](#input\_smb\_domain\_password) | The SMB domain password. | `string` | `""` | no |
| <a name="input_smb_domain_username"></a> [smb\_domain\_username](#input\_smb\_domain\_username) | The SMB domain username. | `string` | `""` | no |
| <a name="input_smb_share_name"></a> [smb\_share\_name](#input\_smb\_share\_name) | The name of the SMB share | `string` | `""` | no |
| <a name="input_smbw_enabled"></a> [smbw\_enabled](#input\_smbw\_enabled) | Enable SMBW protocol. | `bool` | `false` | no |
| <a name="input_source_image_id"></a> [source\_image\_id](#input\_source\_image\_id) | Use weka custom image, ubuntu 20.04 with kernel 5.4 and ofed 5.8-1.1.2.1 | `string` | n/a | yes |
| <a name="input_ssh_public_key"></a> [ssh\_public\_key](#input\_ssh\_public\_key) | The VM public key. If it is not set, the keys are auto-generated. | `string` | n/a | yes |
| <a name="input_subnet_name"></a> [subnet\_name](#input\_subnet\_name) | The subnet names. | `string` | n/a | yes |
| <a name="input_tags_map"></a> [tags\_map](#input\_tags\_map) | A map of tags to assign the same metadata to all resources in the environment. Format: key:value. | `map(string)` | `{}` | no |
| <a name="input_traces_per_frontend"></a> [traces\_per\_frontend](#input\_traces\_per\_frontend) | The number of traces per frontend ionode. Traces are low-level events generated by Weka processes and are used as troubleshooting information for support purposes. Protocol gateways have only frontend ionodes. | `number` | `10` | no |
| <a name="input_vm_username"></a> [vm\_username](#input\_vm\_username) | The user name for logging in to the virtual machines. | `string` | `"weka"` | no |
| <a name="input_vnet_name"></a> [vnet\_name](#input\_vnet\_name) | The virtual network name. | `string` | n/a | yes |
| <a name="input_vnet_rg_name"></a> [vnet\_rg\_name](#input\_vnet\_rg\_name) | Resource group name of vnet | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_vmss_name"></a> [vmss\_name](#output\_vmss\_name) | Protocol gateway vmss name |
| <a name="output_vmss_principal_id"></a> [vmss\_principal\_id](#output\_vmss\_principal\_id) | Protocol gateway vmss principal id |
<!-- END_TF_DOCS -->