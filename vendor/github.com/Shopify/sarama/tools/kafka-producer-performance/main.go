package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	gosync "sync"
	"time"

	"github.com/Shopify/sarama"
	metrics "github.com/rcrowley/go-metrics"
)

var (
	sync = flag.Bool(
		"sync",
		false,
		"Use a synchronous producer.",
	)
	messageLoad = flag.Int(
		"message-load",
		0,
		"REQUIRED: The number of messages to produce to -topic.",
	)
	messageSize = flag.Int(
		"message-size",
		0,
		"REQUIRED: The approximate size (in bytes) of each message to produce to -topic.",
	)
	brokers = flag.String(
		"brokers",
		"",
		"REQUIRED: A comma separated list of broker addresses.",
	)
	topic = flag.String(
		"topic",
		"",
		"REQUIRED: The topic to run the performance test on.",
	)
	partition = flag.Int(
		"partition",
		-1,
		"The partition of -topic to run the performance test on.",
	)
	throughput = flag.Int(
		"throughput",
		0,
		"The maximum number of messages to send per second (0 for no limit).",
	)
	maxMessageBytes = flag.Int(
		"max-message-bytes",
		1000000,
		"The max permitted size of a message.",
	)
	requiredAcks = flag.Int(
		"required-acks",
		1,
		"The required number of acks needed from the broker (-1: all, 0: none, 1: local).",
	)
	timeout = flag.Duration(
		"timeout",
		10*time.Second,
		"The duration the producer will wait to receive -required-acks.",
	)
	partitioner = flag.String(
		"partitioner",
		"roundrobin",
		"The partitioning scheme to use (hash, manual, random, roundrobin).",
	)
	compression = flag.String(
		"compression",
		"none",
		"The compression method to use (none, gzip, snappy, lz4).",
	)
	flushFrequency = flag.Duration(
		"flush-frequency",
		0,
		"The best-effort frequency of flushes.",
	)
	flushBytes = flag.Int(
		"flush-bytes",
		0,
		"The best-effort number of bytes needed to trigger a flush.",
	)
	flushMessages = flag.Int(
		"flush-messages",
		0,
		"The best-effort number of messages needed to trigger a flush.",
	)
	flushMaxMessages = flag.Int(
		"flush-max-messages",
		0,
		"The maximum number of messages the producer will send in a single request.",
	)
	retryMax = flag.Int(
		"retry-max",
		3,
		"The total number of times to retry sending a message.",
	)
	retryBackoff = flag.Duration(
		"retry-backoff",
		100*time.Millisecond,
		"The duration the producer will wait for the cluster to settle between retries.",
	)
	clientID = flag.String(
		"client-id",
		"sarama",
		"The client ID sent with every request to the brokers.",
	)
	channelBufferSize = flag.Int(
		"channel-buffer-size",
		256,
		"The number of events to buffer in internal and external channels.",
	)
	routines = flag.Int(
		"routines",
		1,
		"The number of routines to send the messages from (-sync only).",
	)
	version = flag.String(
		"version",
		"0.8.2.0",
		"The assumed version of Kafka.",
	)
)

func parseCompression(scheme string) sarama.CompressionCodec {
	switch scheme {
	case "none":
		return sarama.CompressionNone
	case "gzip":
		return sarama.CompressionGZIP
	case "snappy":
		return sarama.CompressionSnappy
	case "lz4":
		return sarama.CompressionLZ4
	default:
		printUsageErrorAndExit(fmt.Sprintf("Unknown -compression: %s", scheme))
	}
	panic("should not happen")
}

func parsePartitioner(scheme string, partition int) sarama.PartitionerConstructor {
	if partition < 0 && scheme == "manual" {
		printUsageErrorAndExit("-partition must not be -1 for -partitioning=manual")
	}
	switch scheme {
	case "manual":
		return sarama.NewManualPartitioner
	case "hash":
		return sarama.NewHashPartitioner
	case "random":
		return sarama.NewRandomPartitioner
	case "roundrobin":
		return sarama.NewRoundRobinPartitioner
	default:
		printUsageErrorAndExit(fmt.Sprintf("Unknown -partitioning: %s", scheme))
	}
	panic("should not happen")
}

func parseVersion(version string) sarama.KafkaVersion {
	result, err := sarama.ParseKafkaVersion(version)
	if err != nil {
		printUsageErrorAndExit(fmt.Sprintf("unknown -version: %s", version))
	}
	return result
}

func generateMessages(topic string, partition, messageLoad, messageSize int) []*sarama.ProducerMessage {
	messages := make([]*sarama.ProducerMessage, messageLoad)
	for i := 0; i < messageLoad; i++ {
		payload := make([]byte, messageSize)
		if _, err := rand.Read(payload); err != nil {
			printErrorAndExit(69, "Failed to generate message payload: %s", err)
		}
		messages[i] = &sarama.ProducerMessage{
			Topic:     topic,
			Partition: int32(partition),
			Value:     sarama.ByteEncoder(payload),
		}
	}
	return messages
}

func main() {
	flag.Parse()

	if *brokers == "" {
		printUsageErrorAndExit("-brokers is required")
	}
	if *topic == "" {
		printUsageErrorAndExit("-topic is required")
	}
	if *messageLoad <= 0 {
		printUsageErrorAndExit("-message-load must be greater than 0")
	}
	if *messageSize <= 0 {
		printUsageErrorAndExit("-message-size must be greater than 0")
	}
	if *routines < 1 || *routines > *messageLoad {
		printUsageErrorAndExit("-routines must be greater than 0 and less than or equal to -message-load")
	}

	config := sarama.NewConfig()

	config.Producer.MaxMessageBytes = *maxMessageBytes
	config.Producer.RequiredAcks = sarama.RequiredAcks(*requiredAcks)
	config.Producer.Timeout = *timeout
	config.Producer.Partitioner = parsePartitioner(*partitioner, *partition)
	config.Producer.Compression = parseCompression(*compression)
	config.Producer.Flush.Frequency = *flushFrequency
	config.Producer.Flush.Bytes = *flushBytes
	config.Producer.Flush.Messages = *flushMessages
	config.Producer.Flush.MaxMessages = *flushMaxMessages
	config.Producer.Return.Successes = true
	config.ClientID = *clientID
	config.ChannelBufferSize = *channelBufferSize
	config.Version = parseVersion(*version)

	if err := config.Validate(); err != nil {
		printErrorAndExit(69, "Invalid configuration: %s", err)
	}

	// Print out metrics periodically.
	done := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	go func(ctx context.Context) {
		defer close(done)
		t := time.Tick(5 * time.Second)
		for {
			select {
			case <-t:
				printMetrics(os.Stdout, config.MetricRegistry)
			case <-ctx.Done():
				return
			}
		}
	}(ctx)

	brokers := strings.Split(*brokers, ",")
	if *sync {
		runSyncProducer(*topic, *partition, *messageLoad, *messageSize, *routines,
			config, brokers, *throughput)
	} else {
		runAsyncProducer(*topic, *partition, *messageLoad, *messageSize,
			config, brokers, *throughput)
	}

	cancel()
	<-done

	// Print final metrics.
	printMetrics(os.Stdout, config.MetricRegistry)
}

func runAsyncProducer(topic string, partition, messageLoad, messageSize int,
	config *sarama.Config, brokers []string, throughput int) {
	producer, err := sarama.NewAsyncProducer(brokers, config)
	if err != nil {
		printErrorAndExit(69, "Failed to create producer: %s", err)
	}
	defer func() {
		if err := producer.Close(); err != nil {
			printErrorAndExit(69, "Failed to close producer: %s", err)
		}
	}()

	messages := generateMessages(topic, partition, messageLoad, messageSize)

	messagesDone := make(chan struct{})
	go func() {
		for i := 0; i < messageLoad; i++ {
			select {
			case <-producer.Successes():
			case err = <-producer.Errors():
				printErrorAndExit(69, "%s", err)
			}
		}
		messagesDone <- struct{}{}
	}()

	if throughput > 0 {
		ticker := time.NewTicker(time.Second)
		for _, message := range messages {
			for i := 0; i < throughput; i++ {
				producer.Input() <- message
			}
			<-ticker.C
		}
		ticker.Stop()
	} else {
		for _, message := range messages {
			producer.Input() <- message
		}
	}

	<-messagesDone
	close(messagesDone)
}

func runSyncProducer(topic string, partition, messageLoad, messageSize, routines int,
	config *sarama.Config, brokers []string, throughput int) {
	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		printErrorAndExit(69, "Failed to create producer: %s", err)
	}
	defer func() {
		if err := producer.Close(); err != nil {
			printErrorAndExit(69, "Failed to close producer: %s", err)
		}
	}()

	messages := make([][]*sarama.ProducerMessage, routines)
	for i := 0; i < routines; i++ {
		if i == routines-1 {
			messages[i] = generateMessages(topic, partition, messageLoad/routines+messageLoad%routines, messageSize)
		} else {
			messages[i] = generateMessages(topic, partition, messageLoad/routines, messageSize)
		}
	}

	var wg gosync.WaitGroup
	if throughput > 0 {
		for _, messages := range messages {
			messages := messages
			wg.Add(1)
			go func() {
				ticker := time.NewTicker(time.Second)
				for _, message := range messages {
					for i := 0; i < throughput; i++ {
						_, _, err = producer.SendMessage(message)
						if err != nil {
							printErrorAndExit(69, "Failed to send message: %s", err)
						}
					}
					<-ticker.C
				}
				ticker.Stop()
				wg.Done()
			}()
		}
	} else {
		for _, messages := range messages {
			messages := messages
			wg.Add(1)
			go func() {
				for _, message := range messages {
					_, _, err = producer.SendMessage(message)
					if err != nil {
						printErrorAndExit(69, "Failed to send message: %s", err)
					}
				}
				wg.Done()
			}()
		}
	}
	wg.Wait()
}

func printMetrics(w io.Writer, r metrics.Registry) {
	if r.Get("record-send-rate") == nil || r.Get("request-latency-in-ms") == nil {
		return
	}
	recordSendRate := r.Get("record-send-rate").(metrics.Meter).Snapshot()
	requestLatency := r.Get("request-latency-in-ms").(metrics.Histogram).Snapshot()
	requestLatencyPercentiles := requestLatency.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})
	fmt.Fprintf(w, "%d records sent, %.1f records/sec (%.2f MB/sec), "+
		"%.1f ms avg latency, %.1f ms stddev, %.1f ms 50th, %.1f ms 75th, "+
		"%.1f ms 95th, %.1f ms 99th, %.1f ms 99.9th\n",
		recordSendRate.Count(),
		recordSendRate.RateMean(),
		recordSendRate.RateMean()*float64(*messageSize)/1024/1024,
		requestLatency.Mean(),
		requestLatency.StdDev(),
		requestLatencyPercentiles[0],
		requestLatencyPercentiles[1],
		requestLatencyPercentiles[2],
		requestLatencyPercentiles[3],
		requestLatencyPercentiles[4],
	)
}

func printUsageErrorAndExit(message string) {
	fmt.Fprintln(os.Stderr, "ERROR:", message)
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Available command line options:")
	flag.PrintDefaults()
	os.Exit(64)
}

func printErrorAndExit(code int, format string, values ...interface{}) {
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", fmt.Sprintf(format, values...))
	fmt.Fprintln(os.Stderr)
	os.Exit(code)
}
