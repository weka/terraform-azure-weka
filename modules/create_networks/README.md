## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_azurerm"></a> [azurerm](#requirement\_azurerm) | ~>3.26.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_azurerm"></a> [azurerm](#provider\_azurerm) | ~>3.26.0 |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [azurerm_network_security_group.sg](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/network_security_group) | resource |
| [azurerm_network_security_rule.sg_public_ssh](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/network_security_rule) | resource |
| [azurerm_network_security_rule.sg_weka_ui](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/network_security_rule) | resource |
| [azurerm_private_dns_zone.dns](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/private_dns_zone) | resource |
| [azurerm_private_dns_zone_virtual_network_link.dns_vnet_link](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/private_dns_zone_virtual_network_link) | resource |
| [azurerm_route.private-route](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/route) | resource |
| [azurerm_route.public_route](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/route) | resource |
| [azurerm_route_table.rt](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/route_table) | resource |
| [azurerm_subnet.subnet](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/subnet) | resource |
| [azurerm_subnet_network_security_group_association.sg-association](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/subnet_network_security_group_association) | resource |
| [azurerm_subnet_route_table_association.rt-association](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/subnet_route_table_association) | resource |
| [azurerm_virtual_network.vnet](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/virtual_network) | resource |
| [azurerm_resource_group.rg](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/resource_group) | data source |
| [azurerm_subnet.subnets_data](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/subnet) | data source |
| [azurerm_virtual_network.vnet_data](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/virtual_network) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_address_space"></a> [address\_space](#input\_address\_space) | Address space that is used the virtual network. | `string` | `""` | no |
| <a name="input_prefix"></a> [prefix](#input\_prefix) | Prefix for all resources | `string` | `"weka"` | no |
| <a name="input_private_network"></a> [private\_network](#input\_private\_network) | Should be true to enable private network, defaults to public networking | `bool` | `false` | no |
| <a name="input_rg_name"></a> [rg\_name](#input\_rg\_name) | Name of rg if exist | `string` | `null` | no |
| <a name="input_sg_public_ssh_ips"></a> [sg\_public\_ssh\_ips](#input\_sg\_public\_ssh\_ips) | List of ips to allow ssh on public deployment | `list(string)` | <pre>[<br>  "0.0.0.0/0"<br>]</pre> | no |
| <a name="input_subnet_prefixes"></a> [subnet\_prefixes](#input\_subnet\_prefixes) | List of address prefixes to use for the subnet | `list(string)` | `[]` | no |
| <a name="input_subnets_name_list"></a> [subnets\_name\_list](#input\_subnets\_name\_list) | List of subnets name if existing | `list(string)` | `[]` | no |
| <a name="input_tags_map"></a> [tags\_map](#input\_tags\_map) | Map of tags to set on resources, as key:value | `map(string)` | <pre>{<br>  "creator": "tf",<br>  "env": "dev"<br>}</pre> | no |
| <a name="input_vnet_name"></a> [vnet\_name](#input\_vnet\_name) | Name of vnet if existing | `string` | `null` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_private-dns-zone-name"></a> [private-dns-zone-name](#output\_private-dns-zone-name) | Display private dns zone name |
| <a name="output_rg-name"></a> [rg-name](#output\_rg-name) | Display resource group name |
| <a name="output_sg-id"></a> [sg-id](#output\_sg-id) | Display security group id |
| <a name="output_subnets-name"></a> [subnets-name](#output\_subnets-name) | Display subnets name list |
| <a name="output_vnet-name"></a> [vnet-name](#output\_vnet-name) | Display vnet name |

<!-- BEGIN_TF_DOCS -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_azurerm"></a> [azurerm](#requirement\_azurerm) | ~>3.43.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_azurerm"></a> [azurerm](#provider\_azurerm) | ~>3.43.0 |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [azurerm_network_security_group.sg](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/network_security_group) | resource |
| [azurerm_network_security_rule.sg_public_ssh](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/network_security_rule) | resource |
| [azurerm_network_security_rule.sg_weka_ui](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/network_security_rule) | resource |
| [azurerm_private_dns_zone.blob-dns-zone](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/private_dns_zone) | resource |
| [azurerm_private_dns_zone.dns](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/private_dns_zone) | resource |
| [azurerm_private_dns_zone.keyvault-dns-zone](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/private_dns_zone) | resource |
| [azurerm_private_dns_zone.sites_dns](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/private_dns_zone) | resource |
| [azurerm_private_dns_zone_virtual_network_link.blob_dns_vnet_link](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/private_dns_zone_virtual_network_link) | resource |
| [azurerm_private_dns_zone_virtual_network_link.dns_vnet_link](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/private_dns_zone_virtual_network_link) | resource |
| [azurerm_private_dns_zone_virtual_network_link.keyvault_dns_vnet_link](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/private_dns_zone_virtual_network_link) | resource |
| [azurerm_private_dns_zone_virtual_network_link.sites_dns_vnet_link](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/private_dns_zone_virtual_network_link) | resource |
| [azurerm_subnet.subnet](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/subnet) | resource |
| [azurerm_subnet.subnet-delegation](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/subnet) | resource |
| [azurerm_subnet_network_security_group_association.delegation-sg-association](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/subnet_network_security_group_association) | resource |
| [azurerm_subnet_network_security_group_association.sg-association](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/subnet_network_security_group_association) | resource |
| [azurerm_virtual_network.vnet](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/virtual_network) | resource |
| [azurerm_resource_group.rg](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/resource_group) | data source |
| [azurerm_resource_group.vnet_rg](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/resource_group) | data source |
| [azurerm_subnet.subnets_data](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/subnet) | data source |
| [azurerm_virtual_network.vnet_data](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/virtual_network) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_address_space"></a> [address\_space](#input\_address\_space) | The range of IP addresses the virtual network uses. | `string` | `""` | no |
| <a name="input_blob_dns_zone_link_to_network"></a> [blob\_dns\_zone\_link\_to\_network](#input\_blob\_dns\_zone\_link\_to\_network) | Link private blob DNS to vnet | `bool` | `true` | no |
| <a name="input_create_private_dns_zone"></a> [create\_private\_dns\_zone](#input\_create\_private\_dns\_zone) | Should create private dns zone | `bool` | `true` | no |
| <a name="input_keyvault_dns_zone_link_to_network"></a> [keyvault\_dns\_zone\_link\_to\_network](#input\_keyvault\_dns\_zone\_link\_to\_network) | Link keyvault private DNS zone to network | `bool` | `true` | no |
| <a name="input_keyvault_dns_zone_name"></a> [keyvault\_dns\_zone\_name](#input\_keyvault\_dns\_zone\_name) | Keyvault private DNS zone | `string` | `null` | no |
| <a name="input_prefix"></a> [prefix](#input\_prefix) | The prefix for all the resource names. For example, the prefix for your system name. | `string` | `"weka"` | no |
| <a name="input_private_dns_blob_name"></a> [private\_dns\_blob\_name](#input\_private\_dns\_blob\_name) | privatelink.blob.core.windows.net | `string` | `null` | no |
| <a name="input_private_network"></a> [private\_network](#input\_private\_network) | Determines whether to enable a private or public network. The default is public network. | `bool` | `false` | no |
| <a name="input_rg_name"></a> [rg\_name](#input\_rg\_name) | A predefined resource group in the Azure subscription. | `string` | n/a | yes |
| <a name="input_sg_public_ssh_ips"></a> [sg\_public\_ssh\_ips](#input\_sg\_public\_ssh\_ips) | A list of IP addresses that can use ssh connection with a public network deployment. | `list(string)` | <pre>[<br>  "0.0.0.0/0"<br>]</pre> | no |
| <a name="input_sites_dns_zone_link_to_network"></a> [sites\_dns\_zone\_link\_to\_network](#input\_sites\_dns\_zone\_link\_to\_network) | Link private sites DNS zone to network | `bool` | `true` | no |
| <a name="input_sites_dns_zone_name"></a> [sites\_dns\_zone\_name](#input\_sites\_dns\_zone\_name) | Privatelink azurewebsites DNS zone | `string` | `null` | no |
| <a name="input_subnet_prefixes"></a> [subnet\_prefixes](#input\_subnet\_prefixes) | A list of address prefixes to use for the subnet. | `list(string)` | `[]` | no |
| <a name="input_subnets_delegation_names"></a> [subnets\_delegation\_names](#input\_subnets\_delegation\_names) | List of names of exiting subnets delegation | `list(string)` | `[]` | no |
| <a name="input_subnets_delegation_prefixes"></a> [subnets\_delegation\_prefixes](#input\_subnets\_delegation\_prefixes) | List of subnets delegation enables you to designate a specific subnet for an Azure PaaS service | `list(string)` | `[]` | no |
| <a name="input_subnets_name_list"></a> [subnets\_name\_list](#input\_subnets\_name\_list) | A list of subnet names, if exist. | `list(string)` | `[]` | no |
| <a name="input_tags_map"></a> [tags\_map](#input\_tags\_map) | A map of tags to assign the same metadata to all resources in the environment. Format: key:value. | `map(string)` | <pre>{<br>  "creator": "tf",<br>  "env": "dev"<br>}</pre> | no |
| <a name="input_vnet_name"></a> [vnet\_name](#input\_vnet\_name) | The VNet name, if exists. | `string` | `null` | no |
| <a name="input_vnet_rg_name"></a> [vnet\_rg\_name](#input\_vnet\_rg\_name) | Resource group name of vnet | `string` | `null` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_blob-dns-zone-name"></a> [blob-dns-zone-name](#output\_blob-dns-zone-name) | n/a |
| <a name="output_keyvault-dns-zone-name"></a> [keyvault-dns-zone-name](#output\_keyvault-dns-zone-name) | n/a |
| <a name="output_private-dns-zone-name"></a> [private-dns-zone-name](#output\_private-dns-zone-name) | Displays the private DNS zone name. |
| <a name="output_sg-id"></a> [sg-id](#output\_sg-id) | Displays the security group id. |
| <a name="output_sites-dns-zone-name"></a> [sites-dns-zone-name](#output\_sites-dns-zone-name) | n/a |
| <a name="output_subnets-name"></a> [subnets-name](#output\_subnets-name) | Displays the subnet names list. |
| <a name="output_subnets_delegation_names"></a> [subnets\_delegation\_names](#output\_subnets\_delegation\_names) | n/a |
| <a name="output_vnet-name"></a> [vnet-name](#output\_vnet-name) | Displays the VNet name. |
| <a name="output_vnet_rg_name"></a> [vnet\_rg\_name](#output\_vnet\_rg\_name) | Resource group name of vnet. |
<!-- END_TF_DOCS -->