package redis

import (
	"errors"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/codec"
	"github.com/elastic/beats/libbeat/outputs/outil"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

type redisOut struct {
	beat common.BeatInfo
}

var debugf = logp.MakeDebug("redis")

// Metrics that can retrieved through the expvar web interface.
var (
	redisMetrics = outputs.Metrics.NewRegistry("redis")

	ackedEvents    = monitoring.NewInt(redisMetrics, "events.acked")
	eventsNotAcked = monitoring.NewInt(redisMetrics, "events.not_acked")

	statReadBytes   = monitoring.NewInt(redisMetrics, "read.bytes")
	statWriteBytes  = monitoring.NewInt(redisMetrics, "write.bytes")
	statReadErrors  = monitoring.NewInt(redisMetrics, "read.errors")
	statWriteErrors = monitoring.NewInt(redisMetrics, "write.errors")
)

const (
	defaultWaitRetry    = 1 * time.Second
	defaultMaxWaitRetry = 60 * time.Second
)

func init() {
	outputs.RegisterType("redis", makeRedis)
}

func makeRedis(beat common.BeatInfo, cfg *common.Config) (outputs.Group, error) {
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return outputs.Fail(err)
	}

	var dataType redisDataType
	switch config.DataType {
	case "", "list":
		dataType = redisListType
	case "channel":
		dataType = redisChannelType
	default:
		return outputs.Fail(errors.New("Bad Redis data type"))
	}

	// ensure we have a `key` field in settings
	if cfg.HasField("index") && !cfg.HasField("key") {
		s, err := cfg.String("index", -1)
		if err != nil {
			return outputs.Fail(err)
		}
		if err := cfg.SetString("key", -1, s); err != nil {
			return outputs.Fail(err)
		}
	}
	if !cfg.HasField("index") {
		cfg.SetString("index", -1, beat.Beat)
	}
	if !cfg.HasField("key") {
		cfg.SetString("key", -1, beat.Beat)
	}

	key, err := outil.BuildSelectorFromConfig(cfg, outil.Settings{
		Key:              "key",
		MultiKey:         "keys",
		EnableSingleOnly: true,
		FailEmpty:        true,
	})
	if err != nil {
		return outputs.Fail(err)
	}

	hosts, err := outputs.ReadHostList(cfg)
	if err != nil {
		return outputs.Fail(err)
	}

	tls, err := outputs.LoadTLSConfig(config.TLS)
	if err != nil {
		return outputs.Fail(err)
	}

	transp := &transport.Config{
		Timeout: config.Timeout,
		Proxy:   &config.Proxy,
		TLS:     tls,
		Stats: &transport.IOStats{
			Read:               statReadBytes,
			Write:              statWriteBytes,
			ReadErrors:         statReadErrors,
			WriteErrors:        statWriteErrors,
			OutputsWrite:       outputs.WriteBytes,
			OutputsWriteErrors: outputs.WriteErrors,
		},
	}

	clients := make([]outputs.NetworkClient, len(hosts))
	for i, host := range hosts {
		enc, err := codec.CreateEncoder(config.Codec)
		if err != nil {
			return outputs.Fail(err)
		}

		conn, err := transport.NewClient(transp, "tcp", host, config.Port)
		if err != nil {
			return outputs.Fail(err)
		}

		clients[i] = newClient(conn, config.Timeout,
			config.Password, config.Db, key, dataType, config.Index, enc)
	}

	return outputs.SuccessNet(config.LoadBalance, config.BulkMaxSize, config.MaxRetries, clients)
}
