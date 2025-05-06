---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-module-snyk.html
---

# Snyk module [filebeat-module-snyk]

::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


This is a module for ingesting data from the different Snyk API Endpoints. Currently supports these filesets:

* `vulnerabilities` fileset: Collects all found vulnerabilities for the related organizations and projects
* `audit` fileset: Collects audit logging from Snyk, this can be actions like users, permissions, groups, api access and more.

When you run the module, it performs a few tasks under the hood:

* Sets the default paths to the log files (but don’t worry, you can override the defaults)
* Makes sure each multiline log event gets sent as a single event
* Uses an {{es}} ingest pipeline to parse and process the log lines, shaping the data into a structure suitable for visualizing in Kibana

::::{tip}
Read the [quick start](/reference/filebeat/filebeat-installation-configuration.md) to learn how to configure and run modules.
::::



## Configure the module [configuring-snyk-module]

You can further refine the behavior of the `snyk` module by specifying [variable settings](#snyk-settings) in the `modules.d/snyk.yml` file, or overriding settings at the command line.

You must enable at least one fileset in the module. **Filesets are disabled by default.**


### Variable settings [snyk-settings]

Each fileset has separate variable settings for configuring the behavior of the module. If you don’t specify variable settings, the `snyk` module uses the defaults.

For advanced use cases, you can also override input settings. See [Override input settings](/reference/filebeat/advanced-settings.md).

::::{tip}
When you specify a setting at the command line, remember to prefix the setting with the module name, for example, `snyk.audit.var.paths` instead of `audit.var.paths`.
::::



### `audit` fileset settings [_audit_fileset_settings_6]

::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


To configure access for Filebeat to the Snyk Audit Log API you will have to generate an API access token as described in the [Snyk Documentation](https://snyk.docs.apiary.io/#introduction/authorization)

Example config:

```yaml
- module: snyk
  audit:
    var.input: httpjson
    var.audit_type: organization
    var.audit_id: 1235432-asdfdf-2341234-asdgjhg
    var.interval: 1h
    var.api_token: 53453Sddf8-7fsf-414234gfd-9sdfb7-5asdfh9f8e342
```

There is also multiple optional configuration options that can be used to filter out unwanted content, an example below:

```yaml
- module: snyk
  audit:
    var.input: httpjson
    var.audit_type: organization
    var.audit_id: 1235432-asdfdf-2341234-asdgjhg
    var.interval: 1h
    var.api_token: 53453Sddf8-7fsf-414234gfd-9sdfb7-5asdfh9f8e342
    var.email_address: "test@example.com"
```

**`var.paths`**
:   An array of glob-based paths that specify where to look for the log files. All patterns supported by [Go Glob](https://golang.org/pkg/path/filepath/#Glob) are also supported here. For example, you can use wildcards to fetch all files from a predefined level of subdirectories: `/path/to/log/*/*.log`. This fetches all `.log` files from the subfolders of `/path/to/log`. It does not fetch log files from the `/path/to/log` folder itself. If this setting is left empty, Filebeat will choose log paths based on your operating system.

**`var.first_interval`**
:   How far to look back the first time the module starts, this supports values in full days (24h, 48h etc).

**`var.audit_type`**
:   What audit type to collect, can be either "group" or "organization".

**`var.audit_id`**
:   The ID related to the audit_type. If audit type is group, then this value should be the group ID, or if it is organization it should be the organization ID to collect from.

**`var.api_token`**
:   The API token that is created for a specific user, found in the Snyk management dashboard.

**`var.project_id`**
:   Optional field for filtering, will return only logs for this specific project.

**`var.user_id`**
:   Optional field for filtering, user public ID. Will fetch only audit logs originated from this user’s actions.

**`var.event`**
:   Optional field for filtering, will return only logs for this specific event.

**`var.email_address`**
:   Optional field for filtering, User email address. Will fetch only audit logs originated from this user’s actions.


### Snyk Audit Log ECS Fields [_snyk_audit_log_ecs_fields]

This is a list of Snyk Audit Log fields that are mapped to ECS.

| Snyk Audit log fields | ECS Fields |
| --- | --- |
| groupId | user.group.id |
| userId | user.id |
| event | event.action |
| created | @timestamp |


### `vulnerabilities` fileset settings [_vulnerabilities_fileset_settings]

::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


To configure access for Filebeat to the Snyk Vulnerabilities API you will have to generate an API access token as described in the [Snyk Documentation](https://snyk.docs.apiary.io/#introduction/authorization)

Example config:

```yaml
- module: snyk
  vulnerabilities:
    var.input: httpjson
    var.interval: 24h
    var.api_token: 53453Sddf8-7fsf-414234gfd-9sdfb7-5asdfh9f8e342
    var.orgs:
      - 12354-asdfdf-123543-asdsdfg
      - 76554-jhggfd-654342-hgrfasd
```

There is also multiple optional configuration options that can be used to filter out unwanted content, an example below:

```yaml
- module: snyk
  vulnerabilities:
    var.input: httpjson
    var.interval: 24h
    var.api_token: 53453Sddf8-7fsf-414234gfd-9sdfb7-5asdfh9f8e342
    var.orgs:
      - 12354-asdfdf-123543-asdsdfg
      - 76554-jhggfd-654342-hgrfasd
    var.included_severity:
      - medium
      - high
    var.types:
      - vuln
```

**`var.paths`**
:   An array of glob-based paths that specify where to look for the log files. All patterns supported by [Go Glob](https://golang.org/pkg/path/filepath/#Glob) are also supported here. For example, you can use wildcards to fetch all files from a predefined level of subdirectories: `/path/to/log/*/*.log`. This fetches all `.log` files from the subfolders of `/path/to/log`. It does not fetch log files from the `/path/to/log` folder itself. If this setting is left empty, Filebeat will choose log paths based on your operating system.

**`var.first_interval`**
:   How far to look back the first time the module starts, this supports values in full days (24h, 48h etc).

**`var.api_token`**
:   The API token that is created for a specific user, found in the Snyk management dashboard.

**`var.orgs`**
:   The list of org IDs to filter the results by. One organization ID per line, starting with a - sign

**`var.included_severity`**
:   Optional list of fields for filtering, the severity levels of issues to filter the results by.

**`var.exploit_maturit`**
:   Optional list of fields for filtering, the exploit maturity levels of issues to filter the results by.

**`var.types`**
:   Optional list of fields for filtering, the type of issues to filter the results by.

**`var.languages`**
:   Optional list of fields for filtering, the type of languages to filter the results by.

**`var.identifier`**
:   Optional field for filtering, search term to filter issue name by, or an exact CVE or CWE.

**`var.ignored`**
:   Optional field for filtering, If set to true, only include issues which are ignored, if set to false, only include issues which are not ignored.

**`var.patched`**
:   Optional field for filtering, If set to true, only include issues which are ignored, if set to false, only include issues which are not ignored.

**`var.fixable`**
:   Optional field for filtering, If set to true, only include issues which are ignored, if set to false, only include issues which are not ignored.

**`var.is_fixed`**
:   Optional field for filtering, If set to true, only include issues which are ignored, if set to false, only include issues which are not ignored.

**`var.is_patchable`**
:   Optional field for filtering, If set to true, only include issues which are ignored, if set to false, only include issues which are not ignored.

**`var.is_pinnable`**
:   Optional field for filtering, If set to true, only include issues which are ignored, if set to false, only include issues which are not ignored.

**`var.min_priority_score`**
:   Optional field for filtering, The minimum priority score ranging between 0-1000

**`var.max_priority_score`**
:   Optional field for filtering, The maximum priority score ranging between 0-1000


### Snyk Audit Log ECS Fields [_snyk_audit_log_ecs_fields_2]

This is a list of Snyk Vulnerability fields that are mapped to ECS.

| Snyk Fields | ECS Fields |
| --- | --- |
| issue.description | vulnerability.description |
| issue.identifiers.CVE | vulnerability.id |
| issue.identifiers.ALTERNATIVE | vulnerability.id |
| issue.cvssScore | vulnerability.score.base |
| issue.severity | vulnerability.severity     |
| issue.url | vulnerability.reference |

## Fields [_fields_49]

For a description of each field in the module, see the [exported fields](/reference/filebeat/exported-fields-snyk.md) section.
