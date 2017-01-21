package logstash

// logstash.go defines the logtash plugin (using lumberjack protocol) as being
// registered with all output plugins

import (
	"expvar"
	"time"

	"github.com/elastic/go-lumber/log"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/mode"
	"github.com/elastic/beats/libbeat/outputs/mode/modeutil"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

var debug = logp.MakeDebug("logstash")

// Metrics that can retrieved through the expvar web interface.
var (
	ackedEvents            = expvar.NewInt("libbeat.logstash.published_and_acked_events")
	eventsNotAcked         = expvar.NewInt("libbeat.logstash.published_but_not_acked_events")
	publishEventsCallCount = expvar.NewInt("libbeat.logstash.call_count.PublishEvents")

	statReadBytes   = expvar.NewInt("libbeat.logstash.publish.read_bytes")
	statWriteBytes  = expvar.NewInt("libbeat.logstash.publish.write_bytes")
	statReadErrors  = expvar.NewInt("libbeat.logstash.publish.read_errors")
	statWriteErrors = expvar.NewInt("libbeat.logstash.publish.write_errors")
)

const (
	defaultWaitRetry = 1 * time.Second

	// NOTE: maxWaitRetry has no effect on mode, as logstash client currently does
	// not return ErrTempBulkFailure
	defaultMaxWaitRetry = 60 * time.Second
)

func init() {
	log.Logger = logstashLogger{}

	outputs.RegisterOutputPlugin("logstash", new)
}

func new(beatName string, cfg *common.Config, _ int) (outputs.Outputer, error) {

	if !cfg.HasField("index") {
		cfg.SetString("index", -1, beatName)
	}

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

	tls, err := outputs.LoadTLSConfig(config.TLS)
	if err != nil {
		return err
	}

	transp := &transport.Config{
		Timeout: config.Timeout,
		Proxy:   &config.Proxy,
		TLS:     tls,
		Stats: &transport.IOStats{
			Read:        statReadBytes,
			Write:       statWriteBytes,
			ReadErrors:  statReadErrors,
			WriteErrors: statWriteErrors,
		},
	}

	logp.Info("Max Retries set to: %v", config.MaxRetries)
	m, err := initConnectionMode(cfg, &config, transp)
	if err != nil {
		return err
	}

	lj.mode = m
	lj.index = config.Index

	return nil
}

func initConnectionMode(
	cfg *common.Config,
	config *logstashConfig,
	transp *transport.Config,
) (mode.ConnectionMode, error) {
	sendRetries := config.MaxRetries
	maxAttempts := sendRetries + 1
	if sendRetries < 0 {
		maxAttempts = 0
	}

	settings := modeutil.Settings{
		Failover:     !config.LoadBalance,
		MaxAttempts:  maxAttempts,
		Timeout:      config.Timeout,
		WaitRetry:    defaultWaitRetry,
		MaxWaitRetry: defaultMaxWaitRetry,
	}

	if config.Pipelining == 0 {
		clients, err := modeutil.MakeClients(cfg, makeClientFactory(config, transp))
		if err != nil {
			return nil, err
		}
		return modeutil.NewConnectionMode(clients, settings)
	}

	clients, err := modeutil.MakeAsyncClients(cfg, makeAsyncClientFactory(config, transp))
	if err != nil {
		return nil, err
	}
	return modeutil.NewAsyncConnectionMode(clients, settings)
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

func makeAsyncClientFactory(
	cfg *logstashConfig,
	tcfg *transport.Config,
) modeutil.AsyncClientFactory {
	compressLvl := cfg.CompressionLevel
	maxBulkSz := cfg.BulkMaxSize
	queueSize := cfg.Pipelining - 1
	to := cfg.Timeout

	return func(host string) (mode.AsyncProtocolClient, error) {
		t, err := transport.NewClient(tcfg, "tcp", host, cfg.Port)
		if err != nil {
			return nil, err
		}
		return newAsyncLumberjackClient(t, queueSize, compressLvl, maxBulkSz, to, cfg.Index)
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
	data outputs.Data,
) error {
	return lj.mode.PublishEvent(signaler, opts, data)
}

// BulkPublish implements the BulkOutputer interface pushing a bulk of events
// via lumberjack.
func (lj *logstash) BulkPublish(
	trans op.Signaler,
	opts outputs.Options,
	data []outputs.Data,
) error {
	return lj.mode.PublishEvents(trans, opts, data)
}
