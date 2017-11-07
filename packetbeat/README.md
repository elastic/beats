
# Packetbeat

Packetbeat is an open source network packet analyzer that ships the data to
Elasticsearch. Think of it like a distributed real-time Wireshark with a lot
more analytics features.

The Packetbeat shippers sniff the traffic between your application processes,
parse on the fly protocols like HTTP, MySQL, PostgreSQL, Redis or Thrift and
correlate the messages into transactions.

For each transaction, the shipper inserts a JSON document into Elasticsearch,
where it is stored and indexed. You can then use Kibana to view key metrics and
do ad-hoc queries against the data.

To learn more about Packetbeat, check out <https://www.elastic.co/products/beats/packetbeat>.

## Getting started

Please follow the [getting started](https://www.elastic.co/guide/en/beats/packetbeat/current/packetbeat-getting-started.html)
guide from the docs.

## Documentation

Please visit
[elastic.co](https://www.elastic.co/guide/en/beats/packetbeat/current/index.html) for the
documentation.

## Bugs and feature requests

If you have an issue, please start by opening a topic on the
[forums](https://discuss.elastic.co/c/beats/packetbeat). We'll help you
troubleshoot and work with you on a solution.

If you are sure you found a bug or have a feature request, open an issue on
[Github](https://github.com/elastic/beats/issues).

## Contributions

We love contributions from our community! Please read the
[CONTRIBUTING.md](../CONTRIBUTING.md) file.

## Snapshots

For testing purposes, we generate snapshot builds that you can find [here](https://beats-nightlies.s3.amazonaws.com/index.html?prefix=packetbeat). Please be aware that these are built on top of master and are not meant for production.
