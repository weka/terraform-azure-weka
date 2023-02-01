locals {
  private_ips_list = azurerm_linux_virtual_machine.vm.*.private_ip_address
}

# ================= ui lb =========================== #
resource "azurerm_lb" "ui_lb" {
  name                = "${var.prefix}-${var.cluster_name}-ui-lb"
  resource_group_name = var.rg_name
  location            = data.azurerm_resource_group.rg.location
  sku                 = "Standard"
  frontend_ip_configuration {
    name                          = "${var.prefix}-${var.cluster_name}-ui-lb-frontend"
    subnet_id                     = data.azurerm_subnet.subnets[0].id
    private_ip_address_allocation = "Dynamic"
    private_ip_address_version    = "IPv4"
  }
  tags               = merge(var.tags_map, {"weka_cluster": var.cluster_name})
}

resource "azurerm_lb_backend_address_pool" "ui_lb_backend_pool" {
  name                = "${var.prefix}-${var.cluster_name}-ui-lb-backend-pool"
  loadbalancer_id     = azurerm_lb.ui_lb.id
}

resource "azurerm_lb_probe" "ui_lb_probe" {
  loadbalancer_id     = azurerm_lb.ui_lb.id
  name                = "${var.prefix}-${var.cluster_name}-ui-lb-probe"
  protocol            = "Tcp"
  port                = 14000
  interval_in_seconds = 5
  number_of_probes    = 2
}

resource "azurerm_lb_rule" "ui_lb_rule" {
  loadbalancer_id                = azurerm_lb.ui_lb.id
  name                           = "${var.prefix}-${var.cluster_name}-ui-lb-rule"
  protocol                       = "Tcp"
  frontend_port                  = 14000
  backend_port                   = 14000
  frontend_ip_configuration_name = azurerm_lb.ui_lb.frontend_ip_configuration[0].name
  backend_address_pool_ids       = [azurerm_lb_backend_address_pool.ui_lb_backend_pool.id]
  probe_id                       = azurerm_lb_probe.ui_lb_probe.id
}

resource "azurerm_lb_backend_address_pool_address" "ui_lb_backend_address_pool" {
  count                   = length(local.private_ips_list)
  name                    = "${var.prefix}-${var.cluster_name}-address-pool-${count.index}"
  backend_address_pool_id = azurerm_lb_backend_address_pool.ui_lb_backend_pool.id
  virtual_network_id      = data.azurerm_virtual_network.vnet.id
  ip_address              = local.private_ips_list[count.index]
  depends_on              = [azurerm_linux_virtual_machine.vm,azurerm_lb_backend_address_pool.ui_lb_backend_pool]
}

# ================= backend lb =========================== #
resource "azurerm_lb" "backend-lb" {
  name                = "${var.prefix}-${var.cluster_name}-backend-lb"
  resource_group_name = var.rg_name
  location            = data.azurerm_resource_group.rg.location
  sku                 = "Standard"
  tags                = merge(var.tags_map, {"weka_cluster": var.cluster_name})
  frontend_ip_configuration {
    name                          = "${var.prefix}-${var.cluster_name}-backend-lb-frontend"
    subnet_id                     = data.azurerm_subnet.subnets[0].id
    private_ip_address_allocation = "Dynamic"
    private_ip_address_version    = "IPv4"
  }
}

resource "azurerm_lb_backend_address_pool" "lb_backend_pool" {
  name                = "${var.prefix}-${var.cluster_name}-lb-backend-pool"
  loadbalancer_id     = azurerm_lb.backend-lb.id
}

resource "azurerm_lb_probe" "backend_lb_probe" {
  loadbalancer_id     = azurerm_lb.backend-lb.id
  name                = "${var.prefix}-${var.cluster_name}-lb-probe"
  protocol            = "Tcp"
  port                = 14000
  interval_in_seconds = 5
  number_of_probes    = 2
}

resource "azurerm_lb_rule" "backend_lb_rule" {
  loadbalancer_id                = azurerm_lb.backend-lb.id
  name                           = "${var.prefix}-${var.cluster_name}-backend-lb-rule"
  protocol                       = "Tcp"
  frontend_port                  = 14000
  backend_port                   = 14000
  frontend_ip_configuration_name = azurerm_lb.backend-lb.frontend_ip_configuration[0].name
  backend_address_pool_ids       = [azurerm_lb_backend_address_pool.lb_backend_pool.id]
  probe_id                       = azurerm_lb_probe.backend_lb_probe.id
}

resource "azurerm_lb_backend_address_pool_address" "backend_lb_backend_address_pool" {
  count                   = length(local.private_ips_list)
  name                    = "${var.prefix}-${var.cluster_name}-backend-address-pool-${count.index}"
  backend_address_pool_id = azurerm_lb_backend_address_pool.lb_backend_pool.id
  virtual_network_id      = data.azurerm_virtual_network.vnet.id
  ip_address              = local.private_ips_list[count.index]
  depends_on              = [azurerm_linux_virtual_machine.vm,azurerm_lb_backend_address_pool.lb_backend_pool]
}

# ================== Private DNS  records ========================= #
resource "azurerm_private_dns_a_record" "dns_a_record_ui_lb" {
  name                = "${var.cluster_name}-ui-check"
  zone_name           = var.private_dns_zone_name
  resource_group_name = var.rg_name
  ttl                 = 300
  records             = local.private_ips_list
  tags                = merge(var.tags_map, {"weka_cluster": var.cluster_name})
  depends_on          = [azurerm_lb.ui_lb]
}

resource "azurerm_private_dns_a_record" "dns_a_record_backend_lb" {
  name                = "${var.cluster_name}-backend"
  zone_name           = var.private_dns_zone_name
  resource_group_name = var.rg_name
  ttl                 = 300
  records             = [azurerm_lb.backend-lb.frontend_ip_configuration[0].private_ip_address]
  tags                = merge(var.tags_map, {"weka_cluster": var.cluster_name})
  depends_on          = [azurerm_lb.backend-lb]
}