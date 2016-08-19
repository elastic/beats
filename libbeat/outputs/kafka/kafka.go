package kafka

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Shopify/sarama"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/mode"
	"github.com/elastic/beats/libbeat/outputs/mode/modeutil"
	"github.com/elastic/beats/libbeat/outputs/outil"
)

type kafka struct {
	config kafkaConfig
	topic  outil.Selector

	modeRetry      mode.ConnectionMode
	modeGuaranteed mode.ConnectionMode
}

const (
	defaultWaitRetry = 1 * time.Second

	// NOTE: maxWaitRetry has no effect on mode, as logstash client currently does
	// not return ErrTempBulkFailure
	defaultMaxWaitRetry = 60 * time.Second
)

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

	kafkaVersions = map[string]sarama.KafkaVersion{
		"": sarama.V0_8_2_0,

		"0.8.2.0": sarama.V0_8_2_0,
		"0.8.2.1": sarama.V0_8_2_1,
		"0.8.2.2": sarama.V0_8_2_2,
		"0.8.2":   sarama.V0_8_2_2,
		"0.8":     sarama.V0_8_2_2,

		"0.9.0.0": sarama.V0_9_0_0,
		"0.9.0.1": sarama.V0_9_0_1,
		"0.9.0":   sarama.V0_9_0_1,
		"0.9":     sarama.V0_9_0_1,

		"0.10.0.0": sarama.V0_10_0_0,
		"0.10.0":   sarama.V0_10_0_0,
		"0.10":     sarama.V0_10_0_0,
	}
)

// New instantiates a new kafka output instance.
func New(beatName string, cfg *common.Config, topologyExpire int) (outputs.Outputer, error) {
	output := &kafka{}
	err := output.init(cfg)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func (k *kafka) init(cfg *common.Config) error {
	debugf("initialize kafka output")

	k.config = defaultConfig
	if err := cfg.Unpack(&k.config); err != nil {
		return err
	}

	var err error
	k.topic, err = outil.BuildSelectorFromConfig(cfg, outil.Settings{
		Key:              "topic",
		MultiKey:         "topics",
		EnableSingleOnly: true,
		FailEmpty:        true,
	})
	if err != nil {
		return err
	}

	_, err = newKafkaConfig(&k.config)
	if err != nil {
		return err
	}

	return nil
}

func (k *kafka) initMode(guaranteed bool) (mode.ConnectionMode, error) {
	libCfg, err := newKafkaConfig(&k.config)
	if err != nil {
		return nil, err
	}

	if guaranteed {
		libCfg.Producer.Retry.Max = 1000
	}

	worker := 1
	if k.config.Worker > 1 {
		worker = k.config.Worker
	}

	var clients []mode.AsyncProtocolClient
	hosts := k.config.Hosts
	topic := k.topic
	for i := 0; i < worker; i++ {
		client, err := newKafkaClient(hosts, topic, libCfg)
		if err != nil {
			logp.Err("Failed to create kafka client: %v", err)
			return nil, err
		}
		clients = append(clients, client)
	}

	maxAttempts := 1
	if guaranteed {
		maxAttempts = 0
	}

	mode, err := modeutil.NewAsyncConnectionMode(
		clients,
		false,
		maxAttempts,
		defaultWaitRetry,
		libCfg.Net.WriteTimeout,
		defaultMaxWaitRetry)
	if err != nil {
		logp.Err("Failed to configure kafka connection: %v", err)
		return nil, err
	}
	return mode, nil
}

func (k *kafka) getMode(opts outputs.Options) (mode.ConnectionMode, error) {
	var err error
	guaranteed := opts.Guaranteed || k.config.MaxRetries == -1
	if guaranteed {
		if k.modeGuaranteed == nil {
			k.modeGuaranteed, err = k.initMode(true)
		}
		return k.modeGuaranteed, err
	}

	if k.modeRetry == nil {
		k.modeRetry, err = k.initMode(false)
	}
	return k.modeRetry, err
}

func (k *kafka) Close() error {
	var err error

	if k.modeGuaranteed != nil {
		err = k.modeGuaranteed.Close()
	}
	if k.modeRetry != nil {
		tmp := k.modeRetry.Close()
		if err == nil {
			err = tmp
		}
	}
	return err
}

func (k *kafka) PublishEvent(
	signal op.Signaler,
	opts outputs.Options,
	data outputs.Data,
) error {
	mode, err := k.getMode(opts)
	if err != nil {
		return err
	}
	return mode.PublishEvent(signal, opts, data)
}

func (k *kafka) BulkPublish(
	signal op.Signaler,
	opts outputs.Options,
	data []outputs.Data,
) error {
	mode, err := k.getMode(opts)
	if err != nil {
		return err
	}
	return mode.PublishEvents(signal, opts, data)
}

func newKafkaConfig(config *kafkaConfig) (*sarama.Config, error) {
	k := sarama.NewConfig()

	// configure network level properties
	timeout := config.Timeout
	k.Net.DialTimeout = timeout
	k.Net.ReadTimeout = timeout
	k.Net.WriteTimeout = timeout
	k.Net.KeepAlive = config.KeepAlive
	k.Producer.Timeout = config.BrokerTimeout

	tls, err := outputs.LoadTLSConfig(config.TLS)
	if err != nil {
		return nil, err
	}
	k.Net.TLS.Enable = tls != nil
	k.Net.TLS.Config = tls

	if config.Username != "" {
		k.Net.SASL.Enable = true
		k.Net.SASL.User = config.Username
		k.Net.SASL.Password = config.Password
	}

	// configure metadata update properties
	k.Metadata.Retry.Max = config.Metadata.Retry.Max
	k.Metadata.Retry.Backoff = config.Metadata.Retry.Backoff
	k.Metadata.RefreshFrequency = config.Metadata.RefreshFreq

	// configure producer API properties
	if config.MaxMessageBytes != nil {
		k.Producer.MaxMessageBytes = *config.MaxMessageBytes
	}
	if config.RequiredACKs != nil {
		k.Producer.RequiredAcks = sarama.RequiredAcks(*config.RequiredACKs)
	}

	compressionMode, ok := compressionModes[strings.ToLower(config.Compression)]
	if !ok {
		return nil, fmt.Errorf("Unknown compression mode: %v", config.Compression)
	}
	k.Producer.Compression = compressionMode

	k.Producer.Return.Successes = true // enable return channel for signaling
	k.Producer.Return.Errors = true

	// have retries being handled by libbeat, disable retries in sarama library
	retryMax := config.MaxRetries
	if retryMax < 0 {
		retryMax = 1000
	}
	k.Producer.Retry.Max = retryMax
	// TODO: k.Producer.Retry.Backoff = ?

	// configure per broker go channel buffering
	k.ChannelBufferSize = config.ChanBufferSize

	// configure client ID
	k.ClientID = config.ClientID
	if err := k.Validate(); err != nil {
		logp.Err("Invalid kafka configuration: %v", err)
		return nil, err
	}

	version, ok := kafkaVersions[config.Version]
	if !ok {
		return nil, fmt.Errorf("Unknown/unsupported kafka version: %v", config.Version)
	}
	k.Version = version

	return k, nil
}
