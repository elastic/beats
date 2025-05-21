::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


This metricset periodically fetches JMX metrics from Kafka Broker JMX.


## Compatibility [_compatibility_27]

The module has been tested with Kafka 2.1.1, 2.2.2 and 3.6.0. Other versions are expected to work.


## Usage [_usage_6]

The Broker metricset requires [Jolokia](/reference/metricbeat/metricbeat-module-jolokia.md)to fetch JMX metrics. Refer to the link for instructions about how to use Jolokia.

Note that the Jolokia agent is required to be deployed along with the Kafka JVM application. This can be achieved by using the `KAFKA_OPTS` environment variable when starting the Kafka broker application:

```shell
export KAFKA_OPTS=-javaagent:/opt/jolokia-jvm-1.5.0-agent.jar=port=8779,host=localhost
./bin/kafka-server-start.sh ./config/server.properties
```

Then it will be possible to collect the JMX metrics from `localhost:8779`.
