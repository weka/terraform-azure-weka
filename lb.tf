# ================= ui lb =========================== #
resource "azurerm_lb" "ui_lb" {
  name                = "${var.prefix}-${var.cluster_name}-ui-lb"
  resource_group_name = var.rg_name
  location            = data.azurerm_resource_group.rg.location
  sku                 = "Standard"
  frontend_ip_configuration {
    name                          = "${var.prefix}-${var.cluster_name}-ui-lb-frontend"
    subnet_id                     = data.azurerm_subnet.subnet.id
    private_ip_address_allocation = "Dynamic"
    private_ip_address_version    = "IPv4"
  }
  tags = merge(var.tags_map, { "weka_cluster" : var.cluster_name })
  lifecycle {
    ignore_changes = [tags]
  }
}

resource "azurerm_lb_backend_address_pool" "ui_lb_backend_pool" {
  name            = "${var.prefix}-${var.cluster_name}-ui-lb-backend-pool"
  loadbalancer_id = azurerm_lb.ui_lb.id
  depends_on      = [azurerm_lb.ui_lb]
}

resource "azurerm_lb_probe" "ui_lb_probe" {
  loadbalancer_id     = azurerm_lb.ui_lb.id
  name                = "${var.prefix}-${var.cluster_name}-ui-lb-probe"
  protocol            = "Https"
  request_path        = "/api/v2/healthcheck"
  port                = 14000
  interval_in_seconds = 5
  number_of_probes    = 2
  depends_on          = [azurerm_lb.ui_lb]
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
  depends_on                     = [
    azurerm_lb.ui_lb, azurerm_lb_backend_address_pool.ui_lb_backend_pool, azurerm_lb_probe.ui_lb_probe
  ]
}

# ================= backend lb =========================== #
resource "azurerm_lb" "backend-lb" {
  name                = "${var.prefix}-${var.cluster_name}-backend-lb"
  resource_group_name = var.rg_name
  location            = data.azurerm_resource_group.rg.location
  sku                 = "Standard"
  tags                = merge(var.tags_map, { "weka_cluster" : var.cluster_name })
  frontend_ip_configuration {
    name                          = "${var.prefix}-${var.cluster_name}-backend-lb-frontend"
    subnet_id                     = data.azurerm_subnet.subnet.id
    private_ip_address_allocation = "Dynamic"
    private_ip_address_version    = "IPv4"
  }
  lifecycle {
    ignore_changes = [tags]
  }
}

resource "azurerm_lb_backend_address_pool" "lb_backend_pool" {
  name            = "${var.prefix}-${var.cluster_name}-lb-backend-pool"
  loadbalancer_id = azurerm_lb.backend-lb.id
  depends_on      = [azurerm_lb.backend-lb]
}

resource "azurerm_lb_probe" "backend_lb_probe" {
  loadbalancer_id     = azurerm_lb.backend-lb.id
  name                = "${var.prefix}-${var.cluster_name}-lb-probe"
  protocol            = "Https"
  request_path        = "/api/v2/healthcheck"
  port                = 14000
  interval_in_seconds = 5
  number_of_probes    = 2
  depends_on          = [azurerm_lb.backend-lb]
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
  depends_on                     = [
    azurerm_lb_probe.backend_lb_probe, azurerm_lb_backend_address_pool.lb_backend_pool, azurerm_lb.backend-lb
  ]
}

resource "azurerm_private_dns_a_record" "dns_a_record_backend_lb" {
  count               = var.private_dns_zone_name == null ? 0 : 1
  name                = lower("${var.cluster_name}-backend")
  zone_name           = var.private_dns_zone_name
  resource_group_name = var.rg_name
  ttl                 = 300
  records             = [azurerm_lb.backend-lb.frontend_ip_configuration[0].private_ip_address]
  tags                = merge(var.tags_map, { "weka_cluster" : var.cluster_name })
  depends_on          = [azurerm_lb.backend-lb]
  lifecycle {
    ignore_changes = [tags]
  }
}