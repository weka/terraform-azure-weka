locals {
  supported_regions_filepath = "${abspath(path.root)}/../../${var.supported_regions_file_path}"
  supported_regions_map = {
    for key, val in toset(split("\n", file(local.supported_regions_filepath))) :
    key => val if key != ""
  }
}

data "azurerm_resource_group" "rg" {
  name = var.resource_group_name
}

module "create-sa" {
  source   = "./create_sa"
  for_each = local.supported_regions_map

  rg_name = data.azurerm_resource_group.rg.name
  region  = each.key
}
