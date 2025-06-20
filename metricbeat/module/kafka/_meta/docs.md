:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/kafka/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


This is the Kafka module.

The default metricsets are `consumergroup` and `partition`.

If authorization is configured in the Kafka cluster, the following ACLs are required for the Metricbeat user:

* READ Topic, for the topics to be monitored
* DESCRIBE Group, for the groups to be monitored

For example, if the `stats` user is being used for Metricbeat, to monitor all topics and all consumer groups, ACLS can be granted with the following commands:

```shell
kafka-acls --authorizer-properties zookeeper.connect=localhost:2181 --add --allow-principal User:stats --operation Read --topic '*'
kafka-acls --authorizer-properties zookeeper.connect=localhost:2181 --add --allow-principal User:stats --operation Describe --group '*'
```


## Compatibility [_compatibility_26]

This module is tested with Kafka 0.10.2.1, 1.1.0, 2.1.1, 2.2.2 and 3.6.0.

The Broker, Producer, Consumer metricsets require [Jolokia](/reference/metricbeat/metricbeat-module-jolokia.md) to fetch JMX metrics. Refer to the link for Jolokia’s compatibility notes.


## Usage [_usage_5]

The Broker, Producer, Consumer metricsets require [Jolokia](/reference/metricbeat/metricbeat-module-jolokia.md) to fetch JMX metrics. Refer to those Metricsets' documentation about how to use Jolokia.


## Dashboard [_dashboard_31]

The Kafka module comes with a predefined dashboard. For example:

![metricbeat kafka dashboard](images/metricbeat_kafka_dashboard.png)
