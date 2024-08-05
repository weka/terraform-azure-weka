locals {
  nics = var.frontend_container_cores_num + 1
  script_path = "/tmp/update_aks_node_pool_${var.prefix}_${var.cluster_name}.sh"
}

data "azurerm_resource_group" "rg" {
  name = var.rg_name
}

data "azurerm_subnet" "subnet" {
  name                 = var.subnet_name
  resource_group_name  = var.rg_name
  virtual_network_name = var.vnet_name
}


resource "azurerm_kubernetes_cluster" "k8s" {
  name                      = "${var.prefix}-aks-cluster"
  location                  = data.azurerm_resource_group.rg.location
  resource_group_name       = var.rg_name
  dns_prefix                = "${var.prefix}-aks-dns"
  automatic_channel_upgrade = null
  node_os_channel_upgrade   = "None"
  kubernetes_version        = "1.28.10"

  identity {
    type = "SystemAssigned"
  }

  default_node_pool {
    name                         = "agentpool"
    vm_size                      = var.instance_type
    node_count                   = 3
    enable_auto_scaling          = false
    vnet_subnet_id               = data.azurerm_subnet.subnet.id
    only_critical_addons_enabled = true
    os_sku                       = var.os_sku

  }
  linux_profile {
    admin_username = var.vm_username
    ssh_key {
      key_data = var.ssh_public_key
    }
  }
  network_profile {
    network_plugin      = "azure"
    load_balancer_sku   = "standard"
    network_policy      = "azure"
    network_plugin_mode = "overlay"
  }
  lifecycle {
    ignore_changes = all
  }
  depends_on = [data.azurerm_subnet.subnet]
}

resource "azurerm_kubernetes_cluster_node_pool" "pool" {
  name                  = "clients"
  kubernetes_cluster_id = azurerm_kubernetes_cluster.k8s.id
  vm_size               = var.instance_type
  node_count            = 0
  enable_auto_scaling   = false
  vnet_subnet_id        = data.azurerm_subnet.subnet.id
  os_sku                = var.os_sku
  node_labels = {
    "node" = "weka-client"
  }

  orchestrator_version = azurerm_kubernetes_cluster.k8s.kubernetes_version
  lifecycle {
    ignore_changes = all
  }

  depends_on = [azurerm_kubernetes_cluster.k8s]
}


resource "local_file" "config_yaml" {
  content    = nonsensitive(azurerm_kubernetes_cluster.k8s.kube_config_raw)
  filename   = "/tmp/${var.prefix}-kube-config.yaml"
  depends_on = [azurerm_kubernetes_cluster.k8s]
}

# resource "null_resource" "weka_fs" {
#   triggers = {
#     always_run = timestamp()
#   }
#   provisioner "local-exec" {
#     command = "${path.module}/run.sh ${var.rg_name} ${azurerm_kubernetes_cluster.k8s.name} ${var.key_vault_name} ${var.backend_vmss_name} ${var.subscription_id} ${local.nics} ${var.node_count} ${var.frontend_container_cores_num} ${path.module} \"${var.prefix}-workspace-ml2\""
#   }
#   depends_on = [azurerm_kubernetes_cluster_node_pool.pool]
# }

output "nodes_update_script" {
  value = {
    rg_name = var.rg_name
    aks_cluster_name = azurerm_kubernetes_cluster.k8s.name
    key_vault_name= var.key_vault_name
    backend_vmss_name= var.backend_vmss_name
    subscription_id= var.subscription_id
    nics = local.nics
    node_count = var.node_count
    frontend_container_cores_num= var.frontend_container_cores_num
    yamls_path= path.module
    script_path = local.script_path
  }
}

resource "local_file" "script" {
  filename = local.script_path
    content  = templatefile("${path.module}/run.sh", {
        rg_name = var.rg_name
        aks_cluster_name = azurerm_kubernetes_cluster.k8s.name
        key_vault_name= var.key_vault_name
        backend_vmss_name= var.backend_vmss_name
        subscription_id= var.subscription_id
        nics = local.nics
        node_count = var.node_count
        frontend_container_cores_num= var.frontend_container_cores_num
        yamls_path= path.module
    })
}
