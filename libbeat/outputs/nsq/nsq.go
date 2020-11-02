package nsq

import (
	"time"

	"github.com/Shopify/sarama"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/codec"
	"github.com/elastic/beats/v7/libbeat/outputs/outil"
)

const (
	defaultWaitRetry    = 1 * time.Second
	defaultMaxWaitRetry = 60 * time.Second
	logSelector         = "nsq"
)

func init() {
	sarama.Logger = nsqLogger{log: logp.NewLogger(logSelector)}
	outputs.RegisterType("nsq", makeNsq)
}

func makeNsq(
	_ outputs.IndexManager,
	beat beat.Info,
	observer outputs.Observer,
	cfg *common.Config,
) (outputs.Group, error) {
	log := logp.NewLogger(logSelector)
	log.Debug("initialize nsq output")

	config, err := readConfig(cfg)
	if err != nil {
		return outputs.Fail(err)
	}

	// topic, err := buildTopicSelector(cfg)
	// if err != nil {
	// 	return outputs.Fail(err)
	// }

	// libCfg, err := newSaramaConfig(log, config)
	// if err != nil {
	// 	return outputs.Fail(err)
	// }

	// hosts, err := outputs.ReadHostList(cfg)
	// if err != nil {
	// 	return outputs.Fail(err)
	// }

	codec, err := codec.CreateEncoder(beat, config.Codec)
	if err != nil {
		return outputs.Fail(err)
	}

	// client, err := newNsqClient(observer, config.Nsqd, beat.IndexPrefix, config.Key, topic, codec, libCfg)
	client, err := newNsqClient(observer, config.Nsqd, beat.IndexPrefix, config.Topic, codec,
		config.WriteTimeout, config.DialTimeout)
	if err != nil {
		return outputs.Fail(err)
	}
	// client := newNsqClient(observer, codec, beat.Beat,
	// 	config.Nsqld, config.Topic, config.WriteTimeout, config.DialTimeout)
	return outputs.Success(config.BulkMaxSize, config.MaxRetries, client)
}

func buildTopicSelector(cfg *common.Config) (outil.Selector, error) {
	return outil.BuildSelectorFromConfig(cfg, outil.Settings{
		Key:              "topic",
		MultiKey:         "topics",
		EnableSingleOnly: true,
		FailEmpty:        true,
		Case:             outil.SelectorKeepCase,
	})
}
