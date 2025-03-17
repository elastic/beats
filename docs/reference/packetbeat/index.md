---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/packetbeat-overview.html
  - https://www.elastic.co/guide/en/beats/packetbeat/current/index.html
---

# Packetbeat

Packetbeat is a real-time network packet analyzer that you can use with Elasticsearch to provide an *application monitoring and performance analytics system*. Packetbeat completes the [Beats platform](/reference/index.md) by providing visibility between the servers of your network.

Packetbeat works by capturing the network traffic between your application servers, decoding the application layer protocols (HTTP, MySQL, Redis, and so on), correlating the requests with the responses, and recording the interesting [fields](/reference/packetbeat/exported-fields.md) for each transaction.

Packetbeat can help you easily notice issues with your back-end application, such as bugs or performance problems, and it makes troubleshooting them - and therefore fixing them - much faster.

Packetbeat sniffs the traffic between your servers, parses the application-level protocols on the fly, and correlates the messages into transactions. Currently, Packetbeat supports the following protocols:

* ICMP (v4 and v6)
* DHCP (v4)
* DNS
* HTTP
* AMQP 0.9.1
* Cassandra
* Mysql
* PostgreSQL
* Redis
* Thrift-RPC
* MongoDB
* Memcache
* NFS
* TLS
* SIP/SDP (beta)

Packetbeat can insert the correlated transactions directly into Elasticsearch or into a central queue created with Redis and Logstash.

Packetbeat can run on the same servers as your application processes or on its own servers. When running on dedicated servers, Packetbeat can get the traffic from the switch’s mirror ports or from tapping devices. In such a deployment, there is zero overhead on the monitored application. See [Traffic sniffing](/reference/packetbeat/configuration-interfaces.md) for details.

After decoding the Layer 7 messages, Packetbeat correlates the requests with the responses in what we call *transactions*. For each transaction, Packetbeat inserts a JSON document into Elasticsearch. See the [Exported fields](/reference/packetbeat/exported-fields.md) section for details about which fields are indexed.

The same Elasticsearch and Kibana instances that are used for analysing the network traffic gathered by Packetbeat can be used for analysing the log files gathered by Logstash. This way, you can have network traffic and log analysis in the same system.

Packetbeat is an Elastic [Beat](https://www.elastic.co/beats). It’s based on the `libbeat` framework. For more information, see the [Beats Platform Reference](/reference/index.md).

