---
navigation_title: "Create a _monitoring_ user"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/privileges-to-publish-monitoring.html
---

# Grant privileges and roles needed for monitoring [privileges-to-publish-monitoring]


{{es-security-features}} provides built-in users and roles for monitoring. The privileges and roles needed depend on the method used to collect monitoring data.

::::{admonition} Important note for {{ecloud}} users
:class: important

Built-in users are not available when running our [hosted {{ess}}](https://www.elastic.co/cloud/elasticsearch-service) on {{ecloud}}. To send monitoring data securely, create a monitoring user and grant it the roles described in the following sections.

::::


* If you’re using [internal collection](/reference/metricbeat/monitoring-internal-collection.md) to collect metrics about Metricbeat, {{es-security-features}} provides the `beats_system` [built-in user](docs-content://deploy-manage/users-roles/cluster-or-deployment-auth/built-in-users.md) and `beats_system` [built-in role](elasticsearch://reference/elasticsearch/roles.md) to send monitoring information. You can use the built-in user, if it’s available in your environment, or create a user who has the privileges needed to send monitoring information.

    If you use the `beats_system` user, make sure you set the password.

    If you don’t use the `beats_system` user:

    1. Create a **monitoring role**, called something like `metricbeat_monitoring`, that has the following privileges:

        | Type | Privilege | Purpose |
        | --- | --- | --- |
        | Cluster | `monitor` | Retrieve cluster details (e.g. version) |
        | Index | `create_index` on `.monitoring-beats-*` indices | Create monitoring indices in {{es}} |
        | Index | `create_doc` on `.monitoring-beats-*` indices | Write monitoring events into {{es}} |

    2. Assign the **monitoring role**, along with the following built-in roles, to users who need to monitor Metricbeat:

        | Role | Purpose |
        | --- | --- |
        | `kibana_admin` | Use {{kib}} |
        | `monitoring_user` | Use **Stack Monitoring** in {{kib}} to monitor Metricbeat |

* If you’re [using {{metricbeat}}](/reference/metricbeat/monitoring-metricbeat-collection.md) to collect metrics about Metricbeat, {{es-security-features}} provides the `remote_monitoring_user` [built-in user](docs-content://deploy-manage/users-roles/cluster-or-deployment-auth/built-in-users.md), and the `remote_monitoring_collector` and `remote_monitoring_agent` [built-in roles](elasticsearch://reference/elasticsearch/roles.md) for collecting and sending monitoring information. You can use the built-in user, if it’s available in your environment, or create a user who has the privileges needed to collect and send monitoring information.

    If you use the `remote_monitoring_user` user, make sure you set the password.

    If you don’t use the `remote_monitoring_user` user:

    1. Create a user on the production cluster who will collect and send monitoring information.
    2. Assign the following roles to the user:

        | Role | Purpose |
        | --- | --- |
        | `remote_monitoring_collector` | Collect monitoring metrics from Metricbeat |
        | `remote_monitoring_agent` | Send monitoring data to the monitoring cluster |

    3. Assign the following role to users who will view the monitoring data in {{kib}}:

        | Role | Purpose |
        | --- | --- |
        | `monitoring_user` | Use **Stack Monitoring** in {{kib}} to monitor Metricbeat |


