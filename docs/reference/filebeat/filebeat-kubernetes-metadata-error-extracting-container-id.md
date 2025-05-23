---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-kubernetes-metadata-error-extracting-container-id.html
---

# Error extracting container id while using Kubernetes metadata [filebeat-kubernetes-metadata-error-extracting-container-id]

The `add_kubernetes_metadata` processor might throw the error `Error extracting container id - source value does not contain matcher's logs_path`. There might be some issues with the matchers definitions or the location of `logs_path`. Please verify the Kubernetes pod is healthy.

