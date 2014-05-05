# [Packetbeat](http://packetbeat.com) 

**Open Source Application Monitoring and Packet Tracing system.**

Packetbeat works by sniffing the traffic and analyzing network protocols like HTTP, MySQL and REDIS.

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

The best way to understand the value of a packet monitoring system like Packetbeat is to try it on your own traffic. This quick tutorial will walk you through installing the essential components of the Packetbeat system:

 - The Packetbeat agents for collecting the traffic. You should install these on your servers so that they capture the network traffic.
 - Elasticsearch for storage and search.
 - Kibana for the UI.

For now, you can just install Elasticsearch and Kibana on a single VM or even on your laptop. The only condition is that this machine is accessible from the servers you want to monitor. As you add more agents and your traffic grows, you will want replace the single Elasticsearch instance with a cluster. You will probably also want to automate the installation process. But for now, let's just do the fun part. 

### ElasticSearch

Elasticsearch is a distributed real-time storage, search and analytics engine. It can be used for many purposes, but one context where it excels is indexing streams of semi-structured data, like logs or decoded network packets. 

The binary packages of Elasticsearch have only one dependency: Java. Choose the tab that fits your system (deb for Debian/Ubuntu, rpm for Redhat/Centos/Fedora, binary for the others, including OS X): 

**deb**
```bash
 $ sudo apt-get install default-jre
 $ curl -L -O https://download.elasticsearch.org/elasticsearch/elasticsearch/elasticsearch-1.1.0.deb
 $ sudo dpkg -i elasticsearch-1.1.0.deb
 $ sudo /etc/init.d/elasticsearch start
 ```
 
**rpm**
 ```bash                               
  $ sudo yum install java-1.7.0-openjdk
  $ curl -L -O https://download.elasticsearch.org/elasticsearch/elasticsearch/elasticsearch-1.1.0.rpm
  $ sudo rpm -i elasticsearch-1.1.0.rpm
  $ sudo service elasticsearch start
```

**binary**
```bash
  $ # install Java, e.g. from: https://www.java.com/en/download/manual.jsp
  $ curl -L -O https://download.elasticsearch.org/elasticsearch/elasticsearch/elasticsearch-1.1.0.zip
  $ unzip elasticsearch-1.1.0.zip
  $ cd elasticsearch-1.1.0
  $ ./bin/elasticsearch
```    
To test that the Elasticsearch daemon is up and running, try sending it an HTTP GET on port 9200: 
```
$ curl http://localhost:9200/
    {
      "status" : 200,
      "name" : "Jack Power",
      "version" : {
        "number" : "1.1.0",
        "build_hash" : "2181e113dea80b4a9e31e58e9686658a2d46e363",
        "build_timestamp" : "2014-03-25T15:59:51Z",
        "build_snapshot" : false,
        "lucene_version" : "4.7"
      },
      "tagline" : "You Know, for Search"
    }
    
```
### Kibana UI

### Packetbeat

## Authors

**Monica Sarbu** (monica@packetbeat.com)

## Copyright and license

Copyright Packetbeat 2014. Code released under [the GNU license](LICENSE).




