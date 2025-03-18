---
navigation_title: "Secure"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/auditbeat/current/securing-auditbeat.html
---

# Secure Auditbeat [securing-auditbeat]


The following topics provide information about securing the Auditbeat process and connecting to a cluster that has {{security-features}} enabled.

You can use role-based access control and optionally, API keys to grant Auditbeat users access to secured resources.

* [*Grant users access to secured resources*](/reference/auditbeat/feature-roles.md)
* [*Grant access using API keys*](/reference/auditbeat/beats-api-keys.md).

After privileged users have been created, use authentication to connect to a secured Elastic cluster.

* [*Secure communication with Elasticsearch*](/reference/auditbeat/securing-communication-elasticsearch.md)
* [*Secure communication with Logstash*](/reference/auditbeat/configuring-ssl-logstash.md)

On Linux, Auditbeat can take advantage of secure computing mode to restrict the system calls that a process can issue.

* [*Use Linux Secure Computing Mode (seccomp)*](/reference/auditbeat/linux-seccomp.md)

