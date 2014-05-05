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

For more details about installing ElasticSearch, Kibana and Packetbeat agents you can find under [Getting started](http://packetbeat.com/getstarted).

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
 Kibana is a visualisation application that gets its data from Elasticsearch. It provides a customisable and user-friendly UI in which you can combine various panel types to create your own dashboards. The dashboards can be easily saved, shared and linked.

It is recommended to install Kibana on the same server as Elastic search, but not required.

We have extended Kibana to support new panels specialised in visualising network data. So it is best to download it from [packetbeat GitHub account](http://github.com/packetbeat/packetbeat): 

```bash
 $ curl -L -O https://github.com/packetbeat/kibana/releases/download/v3.0.0-pb/kibana-3.0-packetbeat.tar.gz
 $ tar -xzvf kibana-3.0-packetbeat.tar.gz
```

Kibana is a pure Javascript application running fully in the browser. It doesn't have or need a sever side part like most web applications do. Instead, you only needed a web server to serve the Javascript files and the static resources. For example, you can use python to create a simple web server: 

```bash
  $ cd kibana-3.0-packetbeat
  $ python -m SimpleHTTPServer
    Serving HTTP on 0.0.0.0 port 8000 ...
```
Now point your browser to port 8000 and you should see the Kibana web interface. 


### Packetbeat agent

Now that you have Elasticsearch and Kibana running, I'm sure you are eager to put some data in them. For this, install the Packetbeat agents on your application servers: 

**deb**
```bash
  $ sudo apt-get install lipcap0.8
  $ wget https://github.com/packetbeat/packetbeat/releases/download/v0.1.0/packetbeat_0.1.0-1_amd64.deb
  $ sudo dpkg -i packetbeat_0.1.0-1_amd64.deb
```
**rpm**
```bash
  $ sudo yum install libpcap
  $ wget https://github.com/packetbeat/packetbeat/releases/download/v0.1.0/packetbeat-0.1.0-1.el6.x86_64.rpm
  $ sudo rpm -vi packetbeat-0.1.0-1.el6.x86_64.rpm
```

**binary**
```bash
  $ # install libpcap using your operating system package manager
  $ wget https://github.com/packetbeat/packetbeat/releases/download/v0.1.0/packetbeat-0.1.0-1.el6.x86_64.tar.gz
  $ tar xzvf packetbeat-0.1.0-1.el6.x86_64.tar.gz
```

For the 32 bits packages, fetch packages:


**deb**
```bash
  $ wget https://github.com/packetbeat/packetbeat/releases/download/v0.1.0/packetbeat_0.1.0-1_i386.deb
```

**rpm**
```bash
  $ wget https://github.com/packetbeat/packetbeat/releases/download/v0.1.0/packetbeat-0.1.0-1.el6.i686.rpm
```

**binary**
```bash
  $ wget https://github.com/packetbeat/packetbeat/releases/download/v0.1.0/packetbeat-0.1.0-1.el6.x86_32.tar.gz
```

Before starting the agent, edit the configuration file: 
 
```bash
  $ nano /etc/packetbeat/packetbeat.conf
```
First, set the IP address and port where the agent can find the Elasticsearch installation: 

```
 [elasticsearch]
    # Set the host and port where to find Elasticsearch.
    host = "192.168.1.42"
    port = 9200
```

Select the network interface from which to capture the traffic. Packetbeat supports capturing all messages sent or received by the server on which it is installed. For this, use "any" as the device: 

```
    [interfaces]
    # Select on which network interfaces to sniff. You can use the "any"
    # keyword to sniff on all connected interfaces.
    device = "any"
```

In the next section, you can configure the ports on which Packetbeat can find each protocol. If you use any non-standard ports, add them here. Otherwise, the default values should do just fine. 

```
   [protocols]
    # Configure which protocols to monitor and on which ports are they
    # running. You can disable a given protocol by commenting out its
    # configuration.
      [protocols.http]
      ports = [80, 8080, 8081, 5000, 8002]

      [protocols.mysql]
      ports = [3306]

      [protocols.redis]
      ports = [6379]
```

Finally, you can configure which are the processes that exchange the messages. This part is optional, but configuring the processes enables Packetbeat to not only show you between which servers the traffic is flowing, but also between which processes. It can even show you traffic from between two processes running on the same host. 

```
 [procs]
    # Uncomment the following line to disable the process monitoring.
    # dont_read_from_proc = true

    # Which processes to monitor and how to find them. The processes can
    # be found by searching their command line by a given string.
      [procs.monitored.mysqld]
      cmdline_grep = "mysqld"

      [procs.monitored.nginx]
      cmdline_grep = "nginx"

      [procs.monitored.app]
      cmdline_grep = "gunicorn"
```
With this, you are ready to start the agent: 

**deb**
```bash
  $ /etc/init.d/packetbeat start
```

**rpm**
```bash
   $ service start packetbeat
```

**binary**
```bash
   $ cd packetbeat-0.1.0
   $ ./packetbeat -c /etc/packetbeat/packetbeat.conf
```

### Kibana dashboards

Kibana has about dozen panel types that you can combine into pages to create the UI that is best for you. We have created three sample pages to give you a good start and to demonstrate what is possible.

To load our sample pages, follow these steps: 

```bash
 $ curl -L -O https://github.com/packetbeat/dashboards/archive/v0.1.tar.gz
 $ tar xzvf v0.1.tar.gz
 $ cd dashboards-0.1/
 $ ./load.sh
```

 You should now have the following pages loaded in Kibana. You can switch between them by using the Load menu.

 - Packetbeat Statistics: Contains high-level views like the network topology, the application layer protocols repartition, the response times repartition, and others.
 - Packetbeat Search: This page enables you to do full text searches over the indexed network messages.
 - Packetbeat Query Analysis: This page demonstrates more advanced statistics like the top N slow SQL queries, the database throughput or the most common MySQL errors.


## Authors

**Monica Sarbu** (monica@packetbeat.com)

## Copyright and license

Copyright Packetbeat 2014. Code released under [the GNU license](LICENSE).




