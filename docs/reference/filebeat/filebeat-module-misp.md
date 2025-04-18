---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-module-misp.html
---

# MISP module [filebeat-module-misp]

::::{admonition} Deprecated in 7.14.0.
:class: warning

This module is deprecated. Use the [Threat Intel module](/reference/filebeat/filebeat-module-threatintel.md) instead.
::::


::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


This is a filebeat module for reading threat intel information from the MISP platform ([https://www.circl.lu/doc/misp/](https://www.circl.lu/doc/misp/)). It uses the httpjson input to access the MISP REST API interface.

The configuration in the config.yml file uses the following format:

* var.api_key: specifies the API key to access MISP.
* var.http_request_body: an object containing any parameter that needs to be sent to the search API. Default: `limit: 1000`
* var.url: URL of the MISP REST API, e.g., "http://x.x.x.x/attributes/restSearch"

::::{tip}
Read the [quick start](/reference/filebeat/filebeat-installation-configuration.md) to learn how to configure and run modules.
::::



## Example dashboard [_example_dashboard_13]

This module comes with a sample dashboard. For example:

% TO DO: Use `:class: screenshot`
![kibana misp](images/kibana-misp.png)


## Fields [_fields_30]

For a description of each field in the module, see the [exported fields](/reference/filebeat/exported-fields-misp.md) section.

