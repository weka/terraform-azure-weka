output "IPS" {
  value = module.deploy-weka.*.ip
}

output "SSH-KEY-PATH" {
  value = module.deploy-weka.*.ssh-key-files-path
}

output "DOWNLOAD-SSH-KEYS-COMMAND" {
  value = module.deploy-weka.*.ssh-key-download-blob
}

output "get-cluster-status" {
  value = module.deploy-weka.*.get-cluster-status
  description = "get cluster status command"
}