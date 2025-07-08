`billing` metricset is designed for collecting billing metrics from Google Cloud BigQuery daily cost detail table. BigQuery is a fully-managed, serverless data warehouse. Cloud Billing export to BigQuery enables you to export standard and detailed Google Cloud billing data (such as usage, cost estimates, and pricing data) automatically throughout the day to a BigQuery dataset that you specify. Then you can access your Cloud Billing data from BigQuery for detailed analysis using Metricbeat. Please see [export cloud billing data to BigQuery](https://cloud.google.com/billing/docs/how-to/export-data-bigquery) for more details on how to export billing data.

In BigQuery, Google Cloud daily cost data is categorized into two formats: standard and detailed. Each format is stored within a designated dataset and follows a structured schema for precise cost analysis. For a comprehensive understanding of these formats, consult the [ standard](https://cloud.google.com/billing/docs/how-to/export-data-bigquery-tables/standard-usage#standard-usage-cost-data-schema) and [ detailed](https://cloud.google.com/billing/docs/how-to/export-data-bigquery-tables/detailed-usage#detailed-usage-cost-data-schema) data schema documentation.

For standard usage cost data, set the table pattern format to `gcp_billing_export_v1`. This table pattern is set as the default when no other is specified.

For detailed usage cost data, set the table pattern to `gcp_billing_export_resource_v1`. Detailed tables include the standard fields and additional fields, such as `effective_price`, enabling a more granular view of expenses.


## Metricset-specific configuration notes [_metricset_specific_configuration_notes_12]

* **dataset_id**: (Required) Dataset ID that points to the top-level container which contains the actual billing tables.
* **table_pattern**: (Optional) Daily cost detail billing table name prefix. Default to `gcp_billing_export_v1`.
* **cost_type**: (Optional) The type of cost this line item represents: regular, tax, adjustment, or rounding error. Default to `regular`.


## Configuration example [_configuration_example_20]

```yaml
- module: gcp
  metricsets:
    - billing
  period: 24h
  project_id: "your project id"
  credentials_file_path: "your JSON credentials file path"
  dataset_id: "dataset id"
  table_pattern: "table pattern"
  cost_type: "regular"
```
