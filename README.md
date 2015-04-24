libbeat
=========

The Beats are a collection of daemons that capture and ship data from your
servers to the ELK stack.

The first Beat is [Packetbeat](https://github.com/elastic/packetbeat), a tool
that captures and decodes the traffic between your servers and inserts metadata
about each request-response pair into Elasticsearch. Other Beats will follow,
possible examples being: a Beat for reading and shipping log files (Filebeat), a
Beat for various metrics (Metricbeat), a Beat for real user monitoring
(Rumbeat), etc.

libbeat is the repository containing the Go packages common to all the Beats.
