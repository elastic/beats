---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-module-salesforce.html
applies_to:
  stack: ga
  serverless: ga
---

% This file is generated! See filebeat/scripts/mage/docs.go

# Salesforce module [filebeat-module-salesforce]

::::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/salesforce/index.md).

:::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

:::::


::::::

:::::{note}
The Salesforce module has been completely revamped to use a new dedicated Salesforce input for event collection, replacing the previous HTTPJSON input method. This change brings improved performance and reliability. However, please be aware that this update introduces a breaking change. We believe this is the right time to make this necessary improvement as the previous module was in beta.
:::::


The Salesforce module collects logs from a Salesforce instance using the Salesforce REST API. It supports real-time and historical data collection for various log types including Login, Logout, APEX, and Setup Audit Trail.

The Salesforce module contains the following filesets for collecting different types of logs:

* The `login` fileset collects Login events from the EventLogFile or Objects (real-time).
* The `logout` fileset collects Logout events from the EventLogFile or Objects (real-time).
* The `apex` fileset collects APEX execution logs from the EventLogFile.
* The `setupaudittrail` fileset collects Audit Trails events generated when admins make configuration changes in the org’s Setup area from the Objects (real-time).

| Fileset | EventLogFile | Objects (real-time) |
| --- | --- | --- |
| login | yes | yes |
| logout | yes | yes |
| apex | yes | no |
| setupaudittrail | no | yes |

:::::{important}
The default interval for collecting logs (`var.real_time_interval` or `var.elf_interval`) is 5m/1h. Exercise caution when reducing this interval, as it directly impacts the Salesforce API rate limit of ~1000 calls per hour. Exceeding the limit will result in errors from the Salesforce API. Refer to the [Salesforce API Rate Limit](https://developer.salesforce.com/docs/atlas.en-us.salesforce_app_limits_cheatsheet.meta/salesforce_app_limits_cheatsheet/salesforce_app_limits_platform_api.htm) documentation for more details.

:::::

For high-volume `login` and `logout` real-time collection, the module uses bounded `EventDate` windows during catch-up instead of one large object query. Each run processes up to 12 windows of size `var.real_time_interval`, resumes from persisted progress after restarts, and keeps the SOQL ordering aligned with Salesforce's `EventDate DESC` limitation. If `var.initial_interval` is not set, the first bounded object window falls back to `var.real_time_interval`; set `var.initial_interval` explicitly when you want a deeper first-run catch-up window.

For EventLogFile collection, the built-in filesets keep the initial filter, incremental cursor, and `ORDER BY` aligned on `CreatedDate`, with `Id` as a same-timestamp tie-breaker. Downloaded CSV payloads are streamed row by row to reduce memory pressure. If you override the generated queries, keep those values aligned to avoid gaps or duplicates.

## Fields [_fields]

For a description of each field in the module, see the [exported fields](/reference/filebeat/exported-fields-salesforce.md) section.
