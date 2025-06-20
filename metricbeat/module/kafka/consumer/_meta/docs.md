::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


This metricset periodically fetches JMX metrics from Kafka Consumers implemented in java and expose JMX metrics through jolokia agent.


## Compatibility [_compatibility_28]

The module has been tested with Kafka 2.1.1, 2.2.2 and 3.6.0. Other versions are expected to work.


## Usage [_usage_7]

The Consumer metricset requires [Jolokia](/reference/metricbeat/metricbeat-module-jolokia.md)to fetch JMX metrics. Refer to the link for more information about Jolokia.

Note that the Jolokia agent is required to be deployed along with the JVM application. This can be achieved by using the `KAFKA_OPTS` environment variable when starting the Kafka consumer application:

```shell
export KAFKA_OPTS=-javaagent:/opt/jolokia-jvm-1.5.0-agent.jar=port=8774,host=localhost
./bin/kafka-console-consumer.sh --topic=test --bootstrap-server=localhost:9091
```

Then it will be possible to collect the JMX metrics from `localhost:8774`.
