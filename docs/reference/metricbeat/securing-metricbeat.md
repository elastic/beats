---
navigation_title: "Secure"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/securing-metricbeat.html
---

# Secure Metricbeat [securing-metricbeat]


The following topics provide information about securing the Metricbeat process and connecting to a cluster that has {{security-features}} enabled.

You can use role-based access control and optionally, API keys to grant Metricbeat users access to secured resources.

* [*Grant users access to secured resources*](/reference/metricbeat/feature-roles.md)
* [*Grant access using API keys*](/reference/metricbeat/beats-api-keys.md).

After privileged users have been created, use authentication to connect to a secured Elastic cluster.

* [*Secure communication with Elasticsearch*](/reference/metricbeat/securing-communication-elasticsearch.md)
* [*Secure communication with Logstash*](/reference/metricbeat/configuring-ssl-logstash.md)

On Linux, Metricbeat can take advantage of secure computing mode to restrict the system calls that a process can issue.

* [*Use Linux Secure Computing Mode (seccomp)*](/reference/metricbeat/linux-seccomp.md)

