libBeat
=========

The Beats are a collection of daemons that capture and ship data from your
servers to the ELK stack.

The first Beat is [PacketBeat](https://github.com/elastic/packetbeat), a tool
that captures and decodes the traffic between your servers and inserts metadata
about each request-response pair into Elasticsearch. Other Beats will follow,
possible examples being: a Beat for reading and shipping log files (LogBeat), a
Beat for various metrics (MetricsBeat), a Beat for real user monitoring
(RumBeat), etc.

libBeat is the repository containing the Go packages common to all the Beats.
