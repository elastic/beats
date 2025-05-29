---
navigation_title: "Secure"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/securing-filebeat.html
---

# Secure Filebeat [securing-filebeat]


The following topics provide information about securing the Filebeat process and connecting to a cluster that has {{security-features}} enabled.

You can use role-based access control and optionally, API keys to grant Filebeat users access to secured resources.

* [*Grant users access to secured resources*](/reference/filebeat/feature-roles.md)
* [*Grant access using API keys*](/reference/filebeat/beats-api-keys.md).

After privileged users have been created, use authentication to connect to a secured Elastic cluster.

* [*Secure communication with Elasticsearch*](/reference/filebeat/securing-communication-elasticsearch.md)
* [*Secure communication with Logstash*](/reference/filebeat/configuring-ssl-logstash.md)

On Linux, Filebeat can take advantage of secure computing mode to restrict the system calls that a process can issue.

* [*Use Linux Secure Computing Mode (seccomp)*](/reference/filebeat/linux-seccomp.md)

