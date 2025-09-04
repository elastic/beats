The `vertexai_logs` metricset is designed to collect Vertex AI prompt-response logs from GCP BigQuery. BigQuery is a fully-managed, serverless data warehouse that stores detailed logs of interactions with Vertex AI models.

Vertex AI logs export to BigQuery enables you to export detailed Google Cloud Vertex AI interaction data (such as prompts, responses, model usage, and metadata) automatically to a BigQuery dataset that you specify. Then you can access your Vertex AI logs from BigQuery for detailed analysis and monitoring using Metricbeat. This enables comprehensive tracking of AI model usage, performance monitoring, and cost analysis.

The logs include detailed information about:
- API endpoints and deployed models
- Request and response payloads
- Model versions and API methods used
- OpenTelemetry trace data
- Request metadata and timing information


## Metricset-specific configuration notes [_metricset_specific_configuration_notes_14]

* **table_id**: (Required) Full table identifier in the format `project_id.dataset_id.table_name` that contains the Vertex AI logs data. You can copy this from "Details" tab under 


## Configuration example [_configuration_example_22]

```yaml
- module: gcp
  metricsets:
    - vertexai_logs
  period: 10m
  project_id: "your project id"
  credentials_file_path: "your JSON credentials file path"
  table_id: "your_project.your_dataset.your_vertex_ai_logs_table"
```
