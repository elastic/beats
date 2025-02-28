---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/feature-roles.html
---

# Grant users access to secured resources [feature-roles]

You can use role-based access control to grant users access to secured resources. The roles that you set up depend on your organization’s security requirements and the minimum privileges required to use specific features.

Typically you need the create the following separate roles:

* [setup role](/reference/packetbeat/privileges-to-setup-beats.md) for setting up index templates and other dependencies
* [monitoring role](/reference/packetbeat/privileges-to-publish-monitoring.md) for sending monitoring information
* [writer role](/reference/packetbeat/privileges-to-publish-events.md)  for publishing events collected by Packetbeat
* [reader role](/reference/packetbeat/kibana-user-privileges.md) for {{kib}} users who need to view and create visualizations that access Packetbeat data

{{es-security-features}} provides [built-in roles](elasticsearch://reference/elasticsearch/roles.md) that grant a subset of the privileges needed by Packetbeat users. When possible, use the built-in roles to minimize the affect of future changes on your security strategy.

Instead of using usernames and passwords, roles and privileges can be assigned to API keys to grant access to Elasticsearch resources. See [*Grant access using API keys*](/reference/packetbeat/beats-api-keys.md) for more information.






