package kafka

import (
	"errors"
	"sync"
	"time"

	"github.com/Shopify/sarama"
	gometrics "github.com/rcrowley/go-metrics"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
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

	config := defaultConfig()
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

	libCfg, err := newSaramaConfig(&config)
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
