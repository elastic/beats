output "cloud_id" {
  value       = ec_deployment.default.elasticsearch.cloud_id
  description = "Cloud ID (cloud_id)"
}

output "es_password" {
  value       = ec_deployment.default.elasticsearch_password
  description = "Password (cloud_id)"
  sensitive   = true
}

output "es_username" {
  value       = ec_deployment.default.elasticsearch_username
  description = "Password (cloud_id)"
  sensitive   = true
}

output "es_host" {
  value       = ec_deployment.default.elasticsearch.https_endpoint
  description = ""
}

output "kibana_endpoint" {
  value = ec_deployment.default.kibana.https_endpoint
}
