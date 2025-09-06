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

## Sample Event

Here is a sample event for `vertexai_logs`:

```json
{
  "@timestamp": "2023-12-01T10:30:45.000Z",
  "cloud": {
    "provider": "gcp",
    "project": {
      "id": "my-gcp-project"
    }
  },
  "gcp": {
    "vertexai_logs": {
      "endpoint": "https://us-central1-aiplatform.googleapis.com",
      "deployed_model_id": "1234567890123456789",
      "logging_time": "2023-12-01T10:30:45.000Z",
      "request_id": 98765432101234567,
      "request_payload": ["What is machine learning?"],
      "response_payload": ["Machine learning is a subset of artificial intelligence..."],
      "model": "gemini-2.5-pro",
      "model_version": "1.0",
      "api_method": "generateContent",
      "full_request": {
        "inputs": ["What is machine learning?"],
        "parameters": {
          "temperature": 0.7
        }
      },
      "full_response": {
        "outputs": ["Machine learning is a subset of artificial intelligence..."],
        "usage": {
          "input_tokens": 5,
          "output_tokens": 50
        }
      },
      "metadata": {
        "user_id": "user123",
        "session_id": "session456"
      },
      "otel_log": {
        "trace_id": "abc123def456",
        "span_id": "789ghi012jkl"
      }
    }
  }
}
```