package kafka

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Shopify/sarama"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/mode"
)

type kafka struct {
	mode mode.ConnectionMode
}

func init() {
	sarama.Logger = kafkaLogger{}
	outputs.RegisterOutputPlugin("kafka", New)
}

var debugf = logp.MakeDebug("kafka")

var (
	errNoTopicSet = errors.New("No topic configured")
	errNoHosts    = errors.New("No hosts configured")
)

var (
	compressionModes = map[string]sarama.CompressionCodec{
		"none":   sarama.CompressionNone,
		"no":     sarama.CompressionNone,
		"off":    sarama.CompressionNone,
		"gzip":   sarama.CompressionGZIP,
		"snappy": sarama.CompressionSnappy,
	}
)

func New(cfg *common.Config, topologyExpire int) (outputs.Outputer, error) {
	output := &kafka{}
	err := output.init(cfg)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func (k *kafka) init(cfg *common.Config) error {
	debugf("initialize kafka output")

	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return err
	}

	libCfg, retries, err := newKafkaConfig(&config)
	if err != nil {
		return err
	}

	hosts := config.Hosts
	if len(hosts) < 1 {
		logp.Err("Kafka configuration failed with: %v", errNoHosts)
		return errNoHosts
	}
	debugf("hosts: %v", hosts)

	useType := config.UseType

	topic := config.Topic
	if topic == "" && !useType {
		logp.Err("Kafka configuration failed with: %v", errNoTopicSet)
		return errNoTopicSet
	}

	var clients []mode.AsyncProtocolClient
	worker := 1
	if config.Worker > 1 {
		worker = config.Worker
	}
	for i := 0; i < worker; i++ {
		client, err := newKafkaClient(hosts, topic, useType, libCfg)
		if err != nil {
			logp.Err("Failed to create kafka client: %v", err)
			return err
		}

		clients = append(clients, client)
	}

	mode, err := mode.NewAsyncConnectionMode(
		clients,
		false,
		retries, // retry implemented by kafka client
		libCfg.Producer.Retry.Backoff,
		libCfg.Net.WriteTimeout,
		10*time.Second)
	if err != nil {
		logp.Err("Failed to configure kafka connection: %v", err)
		return err
	}

	k.mode = mode
	return nil
}

func (k *kafka) Close() error {
	return k.mode.Close()
}

func (k *kafka) PublishEvent(
	signal outputs.Signaler,
	opts outputs.Options,
	event common.MapStr,
) error {
	return k.mode.PublishEvent(signal, opts, event)
}

func (k *kafka) BulkPublish(
	signal outputs.Signaler,
	opts outputs.Options,
	event []common.MapStr,
) error {
	return k.mode.PublishEvents(signal, opts, event)
}

func newKafkaConfig(config *kafkaConfig) (*sarama.Config, int, error) {
	k := sarama.NewConfig()
	modeRetries := 1

	// configure network level properties
	timeout := config.Timeout
	k.Net.DialTimeout = timeout
	k.Net.ReadTimeout = timeout
	k.Net.WriteTimeout = timeout
	k.Net.KeepAlive = config.KeepAlive
	k.Producer.Timeout = config.BrokerTimeout

	tls, err := outputs.LoadTLSConfig(config.TLS)
	if err != nil {
		return nil, modeRetries, err
	}
	k.Net.TLS.Enable = tls != nil
	k.Net.TLS.Config = tls

	// TODO: configure metadata level properties
	//       use lib defaults

	// configure producer API properties
	if config.MaxMessageBytes != nil {
		k.Producer.MaxMessageBytes = *config.MaxMessageBytes
	}
	if config.RequiredACKs != nil {
		k.Producer.RequiredAcks = sarama.RequiredAcks(*config.RequiredACKs)
	}

	compressionMode, ok := compressionModes[strings.ToLower(config.Compression)]
	if !ok {
		return nil, modeRetries, fmt.Errorf("Unknown compression mode: %v", config.Compression)
	}
	k.Producer.Compression = compressionMode

	k.Producer.Return.Successes = true // enable return channel for signaling
	k.Producer.Return.Errors = true

	if config.MaxRetries != nil {
		retries := *config.MaxRetries
		if retries < 0 {
			retries = 10
			modeRetries = -1
		}
		k.Producer.Retry.Max = retries
	}

	// configure client ID
	k.ClientID = config.ClientID
	if err := k.Validate(); err != nil {
		logp.Err("Invalid kafka configuration: %v", err)
		return nil, modeRetries, err
	}
	return k, modeRetries, nil
}
