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
| [azurerm_subnet_network_security_group_association.sg_association](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/subnet_network_security_group_association) | resource |
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
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | >= 1.4.6 |
| <a name="requirement_azurerm"></a> [azurerm](#requirement\_azurerm) | ~>3.75.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_azurerm"></a> [azurerm](#provider\_azurerm) | ~>3.75.0 |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [azurerm_nat_gateway.nat_gateway](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/nat_gateway) | resource |
| [azurerm_nat_gateway_public_ip_prefix_association.nat_ip_association](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/nat_gateway_public_ip_prefix_association) | resource |
| [azurerm_network_security_group.sg](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/network_security_group) | resource |
| [azurerm_network_security_rule.sg_public_ssh](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/network_security_rule) | resource |
| [azurerm_network_security_rule.sg_weka_ui](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/network_security_rule) | resource |
| [azurerm_private_dns_zone.dns](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/private_dns_zone) | resource |
| [azurerm_private_dns_zone_virtual_network_link.dns_vnet_link](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/private_dns_zone_virtual_network_link) | resource |
| [azurerm_public_ip_prefix.nat_ip](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/public_ip_prefix) | resource |
| [azurerm_subnet.subnet](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/subnet) | resource |
| [azurerm_subnet_nat_gateway_association.subnet_nat_gateway_association](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/subnet_nat_gateway_association) | resource |
| [azurerm_subnet_network_security_group_association.sg_association](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/subnet_network_security_group_association) | resource |
| [azurerm_virtual_network.vnet](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/virtual_network) | resource |
| [azurerm_resource_group.rg](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/resource_group) | data source |
| [azurerm_resource_group.vnet_rg](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/resource_group) | data source |
| [azurerm_subnet.subnet_data](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/subnet) | data source |
| [azurerm_virtual_network.vnet_data](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/virtual_network) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_address_space"></a> [address\_space](#input\_address\_space) | The range of IP addresses the virtual network uses. | `string` | `"10.0.0.0/16"` | no |
| <a name="input_allow_ssh_cidrs"></a> [allow\_ssh\_cidrs](#input\_allow\_ssh\_cidrs) | Allow port 22, if not provided, i.e leaving the default empty list, the rule will not be included in the SG | `list(string)` | `[]` | no |
| <a name="input_allow_weka_api_cidrs"></a> [allow\_weka\_api\_cidrs](#input\_allow\_weka\_api\_cidrs) | Allow connection to port 14000 on weka backends from specified CIDRs, by default no CIDRs are allowed. All ports (including 14000) are allowed within Vnet | `list(string)` | `[]` | no |
| <a name="input_create_nat_gateway"></a> [create\_nat\_gateway](#input\_create\_nat\_gateway) | NAT needs to be created when no public ip is assigned to the backend, to allow internet access | `bool` | `false` | no |
| <a name="input_prefix"></a> [prefix](#input\_prefix) | The prefix for all the resource names. For example, the prefix for your system name. | `string` | `"weka"` | no |
| <a name="input_private_dns_rg_name"></a> [private\_dns\_rg\_name](#input\_private\_dns\_rg\_name) | The private DNS zone resource group name. Required when private\_dns\_zone\_name is set. | `string` | `""` | no |
| <a name="input_private_dns_zone_name"></a> [private\_dns\_zone\_name](#input\_private\_dns\_zone\_name) | The private DNS zone name. | `string` | `""` | no |
| <a name="input_private_dns_zone_use"></a> [private\_dns\_zone\_use](#input\_private\_dns\_zone\_use) | Determines whether to use private DNS zone. Required for LB dns name. | `bool` | `true` | no |
| <a name="input_rg_name"></a> [rg\_name](#input\_rg\_name) | A predefined resource group in the Azure subscription. | `string` | n/a | yes |
| <a name="input_sg_id"></a> [sg\_id](#input\_sg\_id) | The security group id. | `string` | `""` | no |
| <a name="input_subnet_name"></a> [subnet\_name](#input\_subnet\_name) | Subnet name, if exist. | `string` | `""` | no |
| <a name="input_subnet_prefix"></a> [subnet\_prefix](#input\_subnet\_prefix) | Address prefixes to use for the subnet. | `string` | `"10.0.2.0/24"` | no |
| <a name="input_tags_map"></a> [tags\_map](#input\_tags\_map) | A map of tags to assign the same metadata to all resources in the environment. Format: key:value. | `map(string)` | <pre>{<br>  "creator": "tf",<br>  "env": "dev"<br>}</pre> | no |
| <a name="input_vnet_name"></a> [vnet\_name](#input\_vnet\_name) | The VNet name, if exists. | `string` | `""` | no |
| <a name="input_vnet_rg_name"></a> [vnet\_rg\_name](#input\_vnet\_rg\_name) | Resource group name of vnet | `string` | `""` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_private_dns_rg_name"></a> [private\_dns\_rg\_name](#output\_private\_dns\_rg\_name) | The private DNS zone resource group name. |
| <a name="output_private_dns_zone_name"></a> [private\_dns\_zone\_name](#output\_private\_dns\_zone\_name) | Displays the private DNS zone name. |
| <a name="output_sg_id"></a> [sg\_id](#output\_sg\_id) | Displays the security group id. |
| <a name="output_subnet_name"></a> [subnet\_name](#output\_subnet\_name) | Displays the subnet name list. |
| <a name="output_vnet_name"></a> [vnet\_name](#output\_vnet\_name) | Displays the VNet name. |
| <a name="output_vnet_rg_name"></a> [vnet\_rg\_name](#output\_vnet\_rg\_name) | Resource group name of vnet. |
<!-- END_TF_DOCS -->
