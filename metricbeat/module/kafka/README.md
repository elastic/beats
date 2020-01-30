### Manually testing Kafka modules

Testing Kafka can be tricky, so the purpose of this doc is to document all the steps that one should follow in order to
prepare an environment and manually test Kafka module.

#### Kafka container

In order to have a Kafka instance up and running the best way to go is to use the container that is used by the CI tests.
To bring this container up simply run the tests for Kafka module:

`go test -tags integration ./metricbeat/module/kafka/...`


After the tests have been completed, the Kafka container should be still running. Verify with:

```console
707b50334835    docker.elastic.co/integrations-ci/beats-kafka:2.1.1-2  "/run.sh"        2 minutes ago    Up 2 minutes (healthy)  2181/tcp, 0.0.0.0:32785->8774/tcp, 0.0.0.0:32784->8775/tcp, 0.0.0.0:32783->8779/tcp, 0.0.0.0:32782->9092/tcp  kafka_a035cf4c6889705a_kafka_1
```

In order to identify to which port the Broker is listening on one should check in the logs of the container and find 
the advertised address:

```console
docker logs 707b50334835 > kafka_logs
cat kafka_logs | grep OUTSIDE

advertised.listeners = INSIDE://localhost:9091,OUTSIDE://localhost:32778
listener.security.protocol.map = INSIDE:SASL_PLAINTEXT,OUTSIDE:SASL_PLAINTEXT
listeners = INSIDE://localhost:9091,OUTSIDE://0.0.0.0:9092
advertised.listeners = INSIDE://localhost:9091,OUTSIDE://localhost:32778
listener.security.protocol.map = INSIDE:SASL_PLAINTEXT,OUTSIDE:SASL_PLAINTEXT
listeners = INSIDE://localhost:9091,OUTSIDE://0.0.0.0:9092
```

So here in this example the host we should in the module's config is `localhost:32778`.
Note that this is different between MAC and Linux machines. The above is the case for the MAC machine, and here is how 
the respective address for a LINUX machine should look like:

```console
advertised.listeners = INSIDE://localhost:9091,OUTSIDE://172.26.0.2:9092
listener.security.protocol.map = INSIDE:SASL_PLAINTEXT,OUTSIDE:SASL_PLAINTEXT
listeners = INSIDE://localhost:9091,OUTSIDE://0.0.0.0:9092
advertised.listeners = INSIDE://localhost:9091,OUTSIDE://172.26.0.2:9092
listener.security.protocol.map = INSIDE:SASL_PLAINTEXT,OUTSIDE:SASL_PLAINTEXT
listeners = INSIDE://localhost:9091,OUTSIDE://0.0.0.0:9092
```

So here the advertised addressed to be used in the config is `172.26.0.2:9092`.

This difference comes from here: https://github.com/elastic/beats/blob/master/libbeat/tests/compose/wrapper.go#L137

This was needed before moving the metricbeat docker used in CI to host network, we can maybe remove this now if it complicates things.


#### Configuring Kafka module
In order to configure the Module we will use the advertised addressed to connect to the broker and the credentials
that are also used for the tests 
(see [test config](https://github.com/elastic/beats/blob/6c279ebf2789655725889f37820c959a8f2ea969/metricbeat/module/kafka/consumergroup/consumergroup_integration_test.go#L39)).
Here is how the config should look like (in a MAC):

```yaml
# Kafka metrics collected using the Kafka protocol
- module: kafka
  metricsets:
    - partition
    - consumergroup
  period: 10s
  hosts: ["0.0.0.0:32778"]
  username: stats
  password: test-secret
```


#### Starting extra Producers/Consumers
In order to create more stats for the Kafka Module, one could create more Producer/Consumer pairs (or combinations).
For this we will reuse the scripts that are used withing the Docker container to bring up a Producer/Consumer pair for the testing.
See the [source](https://github.com/elastic/beats/blob/87c49acb60b277a24c60c3956e9b4e23a644bce8/metricbeat/module/kafka/_meta/run.sh#L75).

Here are the commands:

```console
{ while sleep 1; do echo message; done } | KAFKA_OPTS="-Djava.security.auth.login.config=/kafka/bin/jaas-kafka-client-producer.conf" /kafka/bin/kafka-console-producer.sh --topic test2 --broker-list localhost:9091 --producer.config /kafka/bin/sasl-producer.properties
```

Which will start a producer writing a `message` message on topic with name `test2`.

```console
KAFKA_OPTS="-Djava.security.auth.login.config=/kafka/bin/jaas-kafka-client-consumer.conf" /kafka/bin/kafka-console-consumer.sh --topic=test2 --bootstrap-server=localhost:9091 --consumer.config /kafka/bin/sasl-producer.properties
```
Which will start a consumer for `test2` topic.

Note that starting many pairs of them(>4), it might cause the container's crash.

#### JMX data
Kafka Module also includes 3 light modules based on Jolokia Module. These are `broker`, `consumer` and `producer`.

In order to explore the JMX data that are exposed by the container one can use http APIs directly like:

```console
curl -X GET http://0.0.0.0:32783/jolokia/read/kafka.server:\* | jq
```
