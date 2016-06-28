# Metricbeat

Metricbeat takes metrics and statistics from your systems and ships them to elasticsearch or logstash.

**WARNING: Metricbeat is currently still in an experimental phase and under heavy development.**

## Usage

Metricbeat should be installed as local as possible so it can fetch metrics directly from the intended systems. For example if there are multiple MySQL servers, Metricbeat should be installed on each machine if possible instead of a centralised installation.

## Contributions

Contributions of new modules and metricsets to Metricbeat are highly welcome. To guarantee the quality of all metricsets we defined the following requirements for each metricset:

* Unit tests
* Integration tests
* Kibana Dashboards
* Template

Best is to start your own module as its own beat first (see below use it as a library) so you can test it and then start a discussion with our team if it would fit into Metricbeat.

## Use it as library
Metricbeat can also be used as a library so you can implement your own module on top of metricbeat and building your own beat based on it, withouth getting your module into the main repository. This allows to make use of the schedule and interfaces of Metricbeat. A developer guide and how to do this will follow soon.
