package logstash

// logstash.go defines the logtash plugin (using lumberjack protocol) as being
// registered with all output plugins

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/mode"
	"github.com/elastic/beats/libbeat/outputs/mode/modeutil"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

var debug = logp.MakeDebug("logstash")

const (
	defaultWaitRetry = 1 * time.Second

	// NOTE: maxWaitRetry has no effect on mode, as logstash client currently does
	// not return ErrTempBulkFailure
	defaultMaxWaitRetry = 60 * time.Second
)

func init() {
	outputs.RegisterOutputPlugin("logstash", new)
}

func new(cfg *common.Config, _ int) (outputs.Outputer, error) {
	output := &logstash{}
	if err := output.init(cfg); err != nil {
		return nil, err
	}
	return output, nil
}

type logstash struct {
	mode  mode.ConnectionMode
	index string
}

func (lj *logstash) init(cfg *common.Config) error {
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return err
	}

	sendRetries := config.MaxRetries
	maxAttempts := sendRetries + 1
	if sendRetries < 0 {
		maxAttempts = 0
	}

	tls, err := outputs.LoadTLSConfig(config.TLS)
	if err != nil {
		return err
	}

	transp := &transport.Config{
		Timeout: config.Timeout,
		Proxy:   &config.Proxy,
		TLS:     tls,
	}
	clients, err := modeutil.MakeClients(cfg, makeClientFactory(&config, transp))
	if err != nil {
		return err
	}

	logp.Info("Max Retries set to: %v", sendRetries)
	m, err := modeutil.NewConnectionMode(clients, !config.LoadBalance,
		maxAttempts, defaultWaitRetry, config.Timeout, defaultMaxWaitRetry)
	if err != nil {
		return err
	}

	lj.mode = m
	lj.index = config.Index

	return nil
}

func makeClientFactory(
	cfg *logstashConfig,
	tcfg *transport.Config,
) modeutil.ClientFactory {
	compressLvl := cfg.CompressionLevel
	maxBulkSz := cfg.BulkMaxSize
	to := cfg.Timeout

	return func(host string) (mode.ProtocolClient, error) {
		t, err := transport.NewClient(tcfg, "tcp", host, cfg.Port)
		if err != nil {
			return nil, err
		}
		return newLumberjackClient(t, compressLvl, maxBulkSz, to, cfg.Index)
	}
}

func (lj *logstash) Close() error {
	return lj.mode.Close()
}

// TODO: update Outputer interface to support multiple events for batch-like
//       processing (e.g. for filebeat). Batch like processing might reduce
//       send/receive overhead per event for other implementors too.
func (lj *logstash) PublishEvent(
	signaler op.Signaler,
	opts outputs.Options,
	event common.MapStr,
) error {
	return lj.mode.PublishEvent(signaler, opts, event)
}

// BulkPublish implements the BulkOutputer interface pushing a bulk of events
// via lumberjack.
func (lj *logstash) BulkPublish(
	trans op.Signaler,
	opts outputs.Options,
	events []common.MapStr,
) error {
	return lj.mode.PublishEvents(trans, opts, events)
}
