---
navigation_title: "Secure"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/securing-packetbeat.html
---

# Secure Packetbeat [securing-packetbeat]


The following topics provide information about securing the Packetbeat process and connecting to a cluster that has {{security-features}} enabled.

You can use role-based access control and optionally, API keys to grant Packetbeat users access to secured resources.

* [*Grant users access to secured resources*](/reference/packetbeat/feature-roles.md)
* [*Grant access using API keys*](/reference/packetbeat/beats-api-keys.md).

After privileged users have been created, use authentication to connect to a secured Elastic cluster.

* [*Secure communication with Elasticsearch*](/reference/packetbeat/securing-communication-elasticsearch.md)
* [*Secure communication with Logstash*](/reference/packetbeat/configuring-ssl-logstash.md)

On Linux, Packetbeat can take advantage of secure computing mode to restrict the system calls that a process can issue.

* [*Use Linux Secure Computing Mode (seccomp)*](/reference/packetbeat/linux-seccomp.md)

