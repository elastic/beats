package kafka

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Shopify/sarama"
	gometrics "github.com/rcrowley/go-metrics"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
	"github.com/elastic/beats/libbeat/monitoring/adapter"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/codec"
	"github.com/elastic/beats/libbeat/outputs/outil"
)

type kafka struct {
	config kafkaConfig
	topic  outil.Selector

	partitioner sarama.PartitionerConstructor
}

const (
	defaultWaitRetry = 1 * time.Second

	// NOTE: maxWaitRetry has no effect on mode, as logstash client currently does
	// not return ErrTempBulkFailure
	defaultMaxWaitRetry = 60 * time.Second
)

var kafkaMetricsOnce sync.Once
var kafkaMetricsRegistryInstance gometrics.Registry

var debugf = logp.MakeDebug("kafka")

var (
	errNoTopicSet = errors.New("No topic configured")
	errNoHosts    = errors.New("No hosts configured")
)

// TODO: remove me.
// Compat version overwrite for missing versions in sarama
// Public API is compatible between these versions.
var (
	v0_10_2_1 = sarama.V0_10_2_0
	v0_11_0_0 = sarama.V0_10_2_0
)

var (
	compressionModes = map[string]sarama.CompressionCodec{
		"none":   sarama.CompressionNone,
		"no":     sarama.CompressionNone,
		"off":    sarama.CompressionNone,
		"gzip":   sarama.CompressionGZIP,
		"lz4":    sarama.CompressionLZ4,
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
		"0.10.0.1": sarama.V0_10_0_1,
		"0.10.0":   sarama.V0_10_0_1,
		"0.10.1.0": sarama.V0_10_1_0,
		"0.10.1":   sarama.V0_10_1_0,
		"0.10.2.0": sarama.V0_10_2_0,
		"0.10.2.1": v0_10_2_1,
		"0.10.2":   v0_10_2_1,
		"0.10":     v0_10_2_1,

		"0.11.0.0": v0_11_0_0,
		"0.11.0":   v0_11_0_0,
		"0.11":     v0_11_0_0,
	}
)

func init() {
	sarama.Logger = kafkaLogger{}

	reg := gometrics.NewPrefixedRegistry("libbeat.kafka.")

	// Note: registers /debug/metrics handler for displaying all expvar counters
	// TODO: enable
	//exp.Exp(reg)

	kafkaMetricsRegistryInstance = reg

	outputs.RegisterType("kafka", makeKafka)
}

func kafkaMetricsRegistry() gometrics.Registry {
	return kafkaMetricsRegistryInstance
}

func makeKafka(
	beat beat.Info,
	observer outputs.Observer,
	cfg *common.Config,
) (outputs.Group, error) {
	debugf("initialize kafka output")

	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return outputs.Fail(err)
	}

	topic, err := outil.BuildSelectorFromConfig(cfg, outil.Settings{
		Key:              "topic",
		MultiKey:         "topics",
		EnableSingleOnly: true,
		FailEmpty:        true,
	})
	if err != nil {
		return outputs.Fail(err)
	}

	libCfg, err := newKafkaConfig(&config)
	if err != nil {
		return outputs.Fail(err)
	}

	hosts, err := outputs.ReadHostList(cfg)
	if err != nil {
		return outputs.Fail(err)
	}

	codec, err := codec.CreateEncoder(beat, config.Codec)
	if err != nil {
		return outputs.Fail(err)
	}

	client, err := newKafkaClient(observer, hosts, beat.Beat, config.Key, topic, codec, libCfg)
	if err != nil {
		return outputs.Fail(err)
	}

	retry := 0
	if config.MaxRetries < 0 {
		retry = -1
	}
	return outputs.Success(config.BulkMaxSize, retry, client)
}

func newKafkaConfig(config *kafkaConfig) (*sarama.Config, error) {
	partitioner, err := makePartitioner(config.Partition)
	if err != nil {
		return nil, err
	}

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
	if tls != nil {
		k.Net.TLS.Enable = true
		k.Net.TLS.Config = tls.BuildModuleConfig("")
	}

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
		return nil, fmt.Errorf("Unknown compression mode: '%v'", config.Compression)
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

	k.MetricRegistry = kafkaMetricsRegistry()

	k.Producer.Partitioner = partitioner
	k.MetricRegistry = adapter.GetGoMetrics(
		monitoring.Default,
		"libbeat.outputs.kafka",
		adapter.Rename("incoming-byte-rate", "bytes_read"),
		adapter.Rename("outgoing-byte-rate", "bytes_write"),
		adapter.GoMetricsNilify,
	)

	return k, nil
}
