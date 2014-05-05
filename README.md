# [Packetbeat](http://packetbeat.com) 

Open Source Application Monitoring and Packet Tracing system. It works by sniffing the traffic and analyzing network protocols like HTTP, MySQL and REDIS.

To get started, check out <http://packetbeat.com>!

## Table of contents
 - [Integration with ElasticSearch and Kibana](#about)
 - [Bugs and feature requests](#bugs-and-feature-requests)
 - [Install a complete Application Monitoring system](#get-started)
 - [Authors](#authors)
 - [Copyright and license](#copyright-and-license)
 

## Integration with ElasticSearch and Kibana

**100% Open Source, scalable and composable**

Packetbeat is a distributed packet monitoring system that can be used for application performance management. Think of it like a distributed real-time Wireshark with a lot more analytics features.

Packetbeat agents sniff the traffic between your application processes, parse on the fly protocols like HTTP, MySQL or REDIS and correlate the messages into transactions.

For each transaction, the agents insert a JSON document into [Elasticsearch](http://www.elasticsearch.org/overview/elasticsearch/) 
where they are stored and indexed.

The [Kibana](http://www.elasticsearch.org/overview/kibana/) UI application provides advanced visualisations and 
ad-hoc queries. We have extended Kibana with our own panel types for visualising network topologies. 


## Bugs and feature requests

Have a bug or a feature request? Please first check the list of [issues](https://github.com/packetbeat/packetbeat/issues). 
If your problem or idea is not addressed yet,  
[please open a new issue](https://github.com/packetbeat/packetbeat/issues/new).


## Install a complete Application Monitoring system


## Authors

**Monica Sarbu** (monica@packetbeat.com)

## Copyright and license




