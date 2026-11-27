# Serverless endpoints have no port, and Beats would add its default one
# (9200 for Elasticsearch, 5601 for Kibana), so make the HTTPS port explicit.

output "es_host" {
  value       = "${ec_observability_project.default.endpoints.elasticsearch}:443"
  description = "Elasticsearch URL"
}

output "es_username" {
  value       = ec_observability_project.default.credentials.username
  description = "Elasticsearch username"
  sensitive   = true
}

output "es_password" {
  value       = ec_observability_project.default.credentials.password
  description = "Elasticsearch password"
  sensitive   = true
}

output "kibana_endpoint" {
  value       = "${ec_observability_project.default.endpoints.kibana}:443"
  description = "Kibana URL"
}
