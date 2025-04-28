---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-module-google_workspace.html
---

# Google Workspace module [filebeat-module-google_workspace]

:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/google_workspace/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


This is a module for ingesting data from the different Google Workspace audit reports APIs.

::::{tip}
Read the [quick start](/reference/filebeat/filebeat-installation-configuration.md) to learn how to configure and run modules.
::::



## Compatibility [_compatibility_13]

It is compatible with a subset of applications under the [Google Reports API v1](https://developers.google.com/admin-sdk/reports/v1/get-start/getting-started). As of today it supports:

| Google Workspace Service | Description |
| --- | --- | --- |
| SAML [api docs](https://developers.google.com/admin-sdk/reports/v1/appendix/activity/saml) [help](https://support.google.com/a/answer/7007375?hl=en&ref_topic=9027054) | View users’ successful and failed sign-ins to SAML applications. |
| User Accounts [api docs](https://developers.google.com/admin-sdk/reports/v1/appendix/activity/user-accounts) [help](https://support.google.com/a/answer/9022875?hl=en&ref_topic=9027054) | Audit actions carried out by users on their own accounts including password changes, account recovery details and 2-Step Verification enrollment. |
| Login [api docs](https://developers.google.com/admin-sdk/reports/v1/appendix/activity/login) [help](https://support.google.com/a/answer/4580120?hl=en&ref_topic=9027054) | Track user sign-in activity to your domain. |
| Admin [api docs](https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-application-settings) [help](https://support.google.com/a/answer/4579579?hl=en&ref_topic=9027054) | View administrator activity performed within the Google Admin console. |
| Drive [api docs](https://developers.google.com/admin-sdk/reports/v1/appendix/activity/drive) [help](https://support.google.com/a/answer/4579696?hl=en&ref_topic=9027054) | Record user activity within Google Drive including content creation in such as Google Docs, as well as content created elsewhere that your users upload to Drive such as PDFs and Microsoft Word files. |
| Groups [api docs](https://developers.google.com/admin-sdk/reports/v1/appendix/activity/groups) [help](https://support.google.com/a/answer/6270454?hl=en&ref_topic=9027054) | Track changes to groups, group memberships and group messages. |


## Configure the module [_configure_the_module]

In order for Filebeat to ingest data from the Google Reports API you must:

* Have an **administrator account**, as described [here](https://developers.google.com/admin-sdk/reports/v1/guides/prerequisites).
* [Set up a ServiceAccount](https://support.google.com/workspacemigrate/answer/9222993?hl=en) using the administrator account.
* [Set up access to the Admin SDK API](https://developers.google.com/admin-sdk/reports/v1/guides/authorizing) for the ServiceAccount.
* [Enable Domain-Wide Delegation](https://developers.google.com/admin-sdk/reports/v1/guides/delegation) for your ServiceAccount.

This module will make use of the following **oauth2 scope**:

* `https://www.googleapis.com/auth/admin.reports.audit.readonly`

Once you have downloaded your service account credentials as a JSON file, you can set up your module:


#### Configuration options [_configuration_options_42]

```yaml
- module: google_workspace
  saml:
    enabled: true
    var.jwt_file: "./credentials_file.json"
    var.delegated_account: "user@example.com"
  user_accounts:
    enabled: true
    var.jwt_file: "./credentials_file.json"
    var.delegated_account: "user@example.com"
  login:
    enabled: true
    var.jwt_file: "./credentials_file.json"
    var.delegated_account: "user@example.com"
  admin:
    enabled: true
    var.jwt_file: "./credentials_file.json"
    var.delegated_account: "user@example.com"
  drive:
    enabled: true
    var.jwt_file: "./credentials_file.json"
    var.delegated_account: "user@example.com"
  groups:
    enabled: true
    var.jwt_file: "./credentials_file.json"
    var.delegated_account: "user@example.com"
```

Every fileset has the following configuration options:

**`var.jwt_file`**
:   Specifies the path to the JWT credentials file.

**`var.delegated_account`**
:   Email of the admin user used to access the API.

**`var.http_client_timeout`**
:   Duration of the time limit on HTTP requests made by the module. Defaults to `60s`.

**`var.interval`**
:   Duration between requests to the API. Defaults to `2h`.

::::{note}
Google Workspace defaults to a 2 hour polling interval because Google reports can go from some minutes up to 3 days of delay. For more details on this, you can read more [here](https://support.google.com/a/answer/7061566).
::::


**`var.user_key`**
:   Specifies the user key to fetch reports from. Defaults to `all`.

**`var.initial_interval`**
:   It will poll events up to this time period when the module starts. This is to prevent polling too many or repeated events on module restarts. Defaults to `24h`.


### Google Workspace Reports ECS fields [_google_workspace_reports_ecs_fields]

This is a list of Google Workspace Reports fields that are mapped to ECS.

| Google Workspace Reports | ECS Fields |
| --- | --- | --- |
| `items[].id.time` | `@timestamp` |
| `items[].id.uniqueQualifier` | `event.id` |
| `items[].id.applicationName` | `event.provider` |
| `items[].events[].name` | `event.action` |
| `items[].customerId` | `organization.id` |
| `items[].ipAddress` | `source.ip`, `related.ip`, `source.as.*`, `source.geo.*` |
| `items[].actor.email` | `source.user.email`, `source.user.name`, `source.user.domain` |
| `items[].actor.profileId` | `source.user.id` |

These are the common ones to all filesets.


## Fields [_fields_19]

For a description of each field in the module, see the [exported fields](/reference/filebeat/exported-fields-google_workspace.md) section.

