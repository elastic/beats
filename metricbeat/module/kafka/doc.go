/*
Package kafka is a Metricbeat module that contains MetricSets.

Kafka is organised as following

- Topic
- Partition
- Producer
- Consumer
- Consumer Groups
- Broker

Notes
- Topics has a list of partitions
- Each partition has an offset
- Topic can be across brokers
- Each broker has a list of partitions

*/
package kafka
