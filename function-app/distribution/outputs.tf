output "supported_regions" {
  value       = keys(local.supported_regions_map)
  description = "Supported regions list (for release)"
}
