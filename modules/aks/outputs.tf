# output "aks_cluster_username" {
#   value     = azurerm_kubernetes_cluster.k8s.kube_config[0].username
#   sensitive = true
# }
#
# output "aks_host" {
#   value     = azurerm_kubernetes_cluster.k8s.kube_config[0].host
#   sensitive = true
# }
#
# output "kube_config" {
#   value     = nonsensitive(azurerm_kubernetes_cluster.k8s.kube_config_raw)
#   sensitive = false
# }

# output "cluster_id" {
#   value = azurerm_kubernetes_cluster.k8s.id
# }
#
# output "config_file" {
#   value = local_file.config_yaml.filename
# }
#
# output "ml_uri" {
#   value = var.create_ml ? azurerm_machine_learning_workspace.ml[0].discovery_url : null
# }
#
# output "ml_workspace_id" {
#   value = var.create_ml ? azurerm_machine_learning_workspace.ml[0].workspace_id : null
# }
#
# output "aks_client_certificate" {
#   value = azurerm_kubernetes_cluster.k8s.kube_config.0.client_certificate
# }
#
# output "aks_client_key" {
#   value = azurerm_kubernetes_cluster.k8s.kube_config.0.client_key
# }
#
# output "aks_cluster_ca_certificate" {
#   value = azurerm_kubernetes_cluster.k8s.kube_config.0.cluster_ca_certificate
# }
#
# output "aks_rg_name" {
#   value = azurerm_kubernetes_cluster.k8s.resource_group_name
# }
#
# output "aks_weka_node_pool_name" {
#   value = azurerm_kubernetes_cluster_node_pool.pool.name
# }
