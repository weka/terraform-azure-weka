
output "nodes_update_script" {
  value = {
    rg_name                      = var.rg_name
    aks_cluster_name             = azurerm_kubernetes_cluster.k8s.name
    key_vault_name               = var.key_vault_name
    backend_vmss_name            = var.backend_vmss_name
    subscription_id              = var.subscription_id
    nics                         = local.nics
    node_count                   = var.node_count
    frontend_container_cores_num = var.frontend_container_cores_num
    yamls_path                   = path.module
    script_path                  = local.script_path
  }
}
