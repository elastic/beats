libbeat
=========

The Beats are a collection of daemons that capture and ship data from your
servers to Elasticsearch. Read more about Beats on the
[elastic.co](https://stag-www.elastic.co/products/beats) website.

The first Beat is [Packetbeat](https://github.com/elastic/packetbeat), a tool
that captures and decodes the traffic between your servers and inserts metadata
about each request-response pair into Elasticsearch. Other Beats will follow,
possible examples being: a Beat for reading and shipping log files (Filebeat), a
Beat for various OS level metrics (Metricbeat), a Beat for real user monitoring
(Rumbeat), etc.

Libbeat is the repository containing the common Go packages for all Beats.  It
is Apache licensed and actively maintained by the Elastic team.

If you want to create a new project that reads some sort of operational data
and ships it to Elasticsearch, we suggest you make use of this library. Please
open a topic on the [forums](https://discuss.elastic.co/c/beats/libbeat) and
we'll help you get started.
