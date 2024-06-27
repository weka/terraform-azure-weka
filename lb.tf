locals {
  lb_external_ip = var.assign_public_ip ? 1 : 0
}
# ================= ui lb =========================== #
resource "azurerm_public_ip" "ui_ip" {
  count               = var.assign_public_ip ? 1 : 0
  name                = "${var.prefix}-${var.cluster_name}-ui-public-ip"
  location            = local.location
  resource_group_name = local.resource_group_name
  allocation_method   = "Static"
  sku                 = "Standard"
}

resource "azurerm_lb" "ui_lb" {
  count               = var.create_lb ? 1 : 0
  name                = "${var.prefix}-${var.cluster_name}-ui-lb"
  resource_group_name = var.rg_name
  location            = data.azurerm_resource_group.rg.location
  sku                 = "Standard"
  dynamic "frontend_ip_configuration" {
    for_each = range(0, local.lb_external_ip)
    content {
      name                 = "${var.prefix}-${var.cluster_name}-public-ui-frontend"
      public_ip_address_id = azurerm_public_ip.ui_ip[0].id

    }
  }
  dynamic "frontend_ip_configuration" {
    for_each = range(local.lb_external_ip, 1)
    content {
      name                          = "${var.prefix}-${var.cluster_name}-ui-lb-frontend"
      subnet_id                     = data.azurerm_subnet.subnet.id
      private_ip_address_allocation = "Dynamic"
      private_ip_address_version    = "IPv4"
    }
  }

  tags = merge(var.tags_map, { "weka_cluster" : var.cluster_name })
  lifecycle {
    ignore_changes = [tags]
  }
  depends_on = [module.network]
}

resource "azurerm_lb_backend_address_pool" "ui_lb_backend_pool" {
  count           = var.create_lb ? 1 : 0
  name            = "${var.prefix}-${var.cluster_name}-ui-lb-backend-pool"
  loadbalancer_id = azurerm_lb.ui_lb[0].id
  depends_on      = [azurerm_lb.ui_lb]
}

resource "azurerm_lb_probe" "ui_lb_probe" {
  count               = var.create_lb ? 1 : 0
  loadbalancer_id     = azurerm_lb.ui_lb[0].id
  name                = "${var.prefix}-${var.cluster_name}-ui-lb-probe"
  protocol            = "Https"
  request_path        = "/api/v2/ui/healthcheck"
  port                = 14000
  interval_in_seconds = 5
  number_of_probes    = 2
  depends_on          = [azurerm_lb.ui_lb]
}

resource "azurerm_lb_rule" "ui_lb_rule" {
  count                          = var.create_lb ? 1 : 0
  loadbalancer_id                = azurerm_lb.ui_lb[0].id
  name                           = "${var.prefix}-${var.cluster_name}-ui-lb-rule"
  protocol                       = "Tcp"
  frontend_port                  = 14000
  backend_port                   = 14000
  frontend_ip_configuration_name = azurerm_lb.ui_lb[0].frontend_ip_configuration[0].name
  backend_address_pool_ids       = [azurerm_lb_backend_address_pool.ui_lb_backend_pool[0].id]
  probe_id                       = azurerm_lb_probe.ui_lb_probe[0].id
  depends_on = [
    azurerm_lb.ui_lb, azurerm_lb_backend_address_pool.ui_lb_backend_pool, azurerm_lb_probe.ui_lb_probe
  ]
}

# ================= backend lb =========================== #
resource "azurerm_public_ip" "backend_ip" {
  count               = var.assign_public_ip ? 1 : 0
  name                = "${var.prefix}-${var.cluster_name}-backend-public-ip"
  location            = local.location
  resource_group_name = local.resource_group_name
  allocation_method   = "Static"
  sku                 = "Standard"
}

resource "azurerm_lb" "backend_lb" {
  count               = var.create_lb ? 1 : 0
  name                = "${var.prefix}-${var.cluster_name}-backend-lb"
  resource_group_name = var.rg_name
  location            = data.azurerm_resource_group.rg.location
  sku                 = "Standard"
  tags                = merge(var.tags_map, { "weka_cluster" : var.cluster_name })
  dynamic "frontend_ip_configuration" {
    for_each = range(0, local.lb_external_ip)
    content {
      name                 = "${var.prefix}-${var.cluster_name}-public-backend-frontend"
      public_ip_address_id = azurerm_public_ip.backend_ip[0].id
    }
  }
  dynamic "frontend_ip_configuration" {
    for_each = range(local.lb_external_ip, 1)
    content {
      name                          = "${var.prefix}-${var.cluster_name}-backend-lb-frontend"
      subnet_id                     = data.azurerm_subnet.subnet.id
      private_ip_address_allocation = "Dynamic"
      private_ip_address_version    = "IPv4"
    }
  }
  lifecycle {
    ignore_changes = [tags]
  }
  depends_on = [module.network]
}

resource "azurerm_lb_backend_address_pool" "lb_backend_pool" {
  count           = var.create_lb ? 1 : 0
  name            = "${var.prefix}-${var.cluster_name}-lb-backend-pool"
  loadbalancer_id = azurerm_lb.backend_lb[0].id
  depends_on      = [azurerm_lb.backend_lb]
}

resource "azurerm_lb_probe" "backend_lb_probe" {
  count               = var.create_lb ? 1 : 0
  loadbalancer_id     = azurerm_lb.backend_lb[0].id
  name                = "${var.prefix}-${var.cluster_name}-lb-probe"
  protocol            = "Http"
  request_path        = "/api/v2/healthcheck"
  port                = 14000
  interval_in_seconds = 5
  number_of_probes    = 2
  depends_on          = [azurerm_lb.backend_lb]
}

resource "azurerm_lb_rule" "backend_lb_rule" {
  count                          = var.create_lb ? 1 : 0
  loadbalancer_id                = azurerm_lb.backend_lb[0].id
  name                           = "${var.prefix}-${var.cluster_name}-backend-lb-rule"
  protocol                       = "Tcp"
  frontend_port                  = 14000
  backend_port                   = 14000
  frontend_ip_configuration_name = azurerm_lb.backend_lb[0].frontend_ip_configuration[0].name
  backend_address_pool_ids       = [azurerm_lb_backend_address_pool.lb_backend_pool[0].id]
  probe_id                       = azurerm_lb_probe.backend_lb_probe[0].id
  depends_on = [
    azurerm_lb_probe.backend_lb_probe, azurerm_lb_backend_address_pool.lb_backend_pool, azurerm_lb.backend_lb
  ]
}

resource "azurerm_private_dns_a_record" "dns_a_record_backend_lb" {
  count               = var.create_lb ? 1 : 0
  name                = lower("${var.cluster_name}-backend")
  zone_name           = local.private_dns_zone_name
  resource_group_name = local.private_dns_rg_name
  ttl                 = 300
  records             = var.assign_public_ip ? [azurerm_public_ip.backend_ip[0].ip_address] : [azurerm_lb.backend_lb[0].frontend_ip_configuration[0].private_ip_address]
  tags                = merge(var.tags_map, { "weka_cluster" : var.cluster_name })
  depends_on          = [azurerm_lb.backend_lb, module.network]
  lifecycle {
    ignore_changes = [tags]
  }
}
