package logstash

// logstash.go defines the logtash plugin (using lumberjack protocol) as being registered with all
// output plugins

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/mode"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

var debug = logp.MakeDebug("logstash")

func init() {
	outputs.RegisterOutputPlugin("logstash", New)
}

func New(cfg *common.Config, _ int) (outputs.Outputer, error) {
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

var waitRetry = time.Duration(1) * time.Second

// NOTE: maxWaitRetry has no effect on mode, as logstash client currently does not return ErrTempBulkFailure
var maxWaitRetry = time.Duration(60) * time.Second

func (lj *logstash) init(cfg *common.Config) error {
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return err
	}

	useTLS := (config.TLS != nil)
	sendRetries := config.MaxRetries
	maxAttempts := sendRetries + 1
	if sendRetries < 0 {
		maxAttempts = 0
	}

	transp := &transport.Config{
		Timeout: config.Timeout,
		Proxy:   &config.Proxy,
	}
	if useTLS {
		var err error
		transp.TLS, err = outputs.LoadTLSConfig(config.TLS)
		if err != nil {
			return err
		}
	}

	clients, err := mode.MakeClients(cfg,
		makeClientFactory(&config, func(host string) (*transport.Client, error) {
			return transport.NewClient(transp, "tcp", host, config.Port)
		}))
	if err != nil {
		return err
	}

	logp.Info("Max Retries set to: %v", sendRetries)
	m, err := mode.NewConnectionMode(clients, !config.LoadBalance,
		maxAttempts, waitRetry, config.Timeout, maxWaitRetry)
	if err != nil {
		return err
	}

	lj.mode = m
	lj.index = config.Index

	return nil
}

func makeClientFactory(
	config *logstashConfig,
	makeTransp func(string) (*transport.Client, error),
) func(string) (mode.ProtocolClient, error) {
	return func(host string) (mode.ProtocolClient, error) {
		transp, err := makeTransp(host)
		if err != nil {
			return nil, err
		}
		return newLumberjackClient(transp,
			config.CompressionLevel, config.BulkMaxSize, config.Timeout)
	}
}

func (lj *logstash) Close() error {
	return lj.mode.Close()
}

// TODO: update Outputer interface to support multiple events for batch-like
//       processing (e.g. for filebeat). Batch like processing might reduce
//       send/receive overhead per event for other implementors too.
func (lj *logstash) PublishEvent(
	signaler outputs.Signaler,
	opts outputs.Options,
	event common.MapStr,
) error {
	lj.addMeta(event)
	return lj.mode.PublishEvent(signaler, opts, event)
}

// BulkPublish implements the BulkOutputer interface pushing a bulk of events
// via lumberjack.
func (lj *logstash) BulkPublish(
	trans outputs.Signaler,
	opts outputs.Options,
	events []common.MapStr,
) error {
	for _, event := range events {
		lj.addMeta(event)
	}
	return lj.mode.PublishEvents(trans, opts, events)
}

// addMeta adapts events to be compatible with logstash forwarer messages by renaming
// the "message" field to "line". The lumberjack server in logstash will
// decode/rename the "line" field into "message".
func (lj *logstash) addMeta(event common.MapStr) {
	// add metadata for indexing
	event["@metadata"] = common.MapStr{
		"beat": lj.index,
		"type": event["type"].(string),
	}
}
