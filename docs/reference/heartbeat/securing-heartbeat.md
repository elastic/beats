---
navigation_title: "Secure"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/securing-heartbeat.html
---

# Secure Heartbeat [securing-heartbeat]


The following topics provide information about securing the Heartbeat process and connecting to a cluster that has {{security-features}} enabled.

You can use role-based access control and optionally, API keys to grant Heartbeat users access to secured resources.

* [*Grant users access to secured resources*](/reference/heartbeat/feature-roles.md)
* [*Grant access using API keys*](/reference/heartbeat/beats-api-keys.md).

After privileged users have been created, use authentication to connect to a secured Elastic cluster.

* [*Secure communication with Elasticsearch*](/reference/heartbeat/securing-communication-elasticsearch.md)
* [*Secure communication with Logstash*](/reference/heartbeat/configuring-ssl-logstash.md)

On Linux, Heartbeat can take advantage of secure computing mode to restrict the system calls that a process can issue.

* [*Use Linux Secure Computing Mode (seccomp)*](/reference/heartbeat/linux-seccomp.md)

