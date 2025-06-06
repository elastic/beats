::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


The `carbon` metricset is designed to collect Carbon Footprint data from GCP BigQuery monthly cost detail table. BigQuery is a fully-managed, serverless data warehouse.

Cloud Carbon export to BigQuery enables you to export detailed Google Cloud carbon footprint data (such as carbon produced by tier and service) automatically throughout the month to a BigQuery dataset that you specify. Then you can access your Cloud Carbon data from BigQuery for detailed analysis using Metricbeat. Please see [export carbon footprint data to BigQuery](https://cloud.google.com/carbon-footprint/docs/export) for more details on how to export carbon footprint data.


## Metricset-specific configuration notes [_metricset_specific_configuration_notes_13]

* **dataset_id**: (Required) Dataset ID that points to the top-level container which contains the actual carbon footprint tables.
* **table_pattern**: (Optional) The name of the table where carbon footprint data is stored. Default to `carbon_footprint`.


## Configuration example [_configuration_example_21]

```yaml
- module: gcp
  metricsets:
    - carbon
  period: 24h
  project_id: "your project id"
  credentials_file_path: "your JSON credentials file path"
  dataset_id: "dataset id"
  table_name: "table name"
```
