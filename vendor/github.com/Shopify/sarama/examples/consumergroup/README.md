# Consumergroup example

This example shows you how to use the Sarama consumer group consumer. The example simply starts consuming the given Kafka topics and logs the consumed messages.

```bash
$ go run main.go -brokers="127.0.0.1:9092" -topics="sarama" -group="example"
```