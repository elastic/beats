package logstash

// logstash.go defines the logtash plugin (using lumberjack protocol) as being registered with all
// output plugins

import (
	"crypto/tls"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/mode"
)

var debug = logp.MakeDebug("logstash")

func init() {
	outputs.RegisterOutputPlugin("logstash", logstashOutputPlugin{})
}

type logstashOutputPlugin struct{}

func (p logstashOutputPlugin) NewOutput(
	config *outputs.MothershipConfig,
	topologyExpire int,
) (outputs.Outputer, error) {
	output := &logstash{}
	err := output.init(*config, topologyExpire)
	if err != nil {
		return nil, err
	}
	return output, nil
}

type logstash struct {
	mode  mode.ConnectionMode
	index string
}

const (
	logstashDefaultPort = 10200

	logstashDefaultTimeout  = 30 * time.Second
	defaultSendRetries      = 3
	defaultMaxWindowSize    = 2048
	defaultCompressionLevel = 3
)

var waitRetry = time.Duration(1) * time.Second

// NOTE: maxWaitRetry has no effect on mode, as logstash client currently does not return ErrTempBulkFailure
var maxWaitRetry = time.Duration(60) * time.Second

func (lj *logstash) init(
	config outputs.MothershipConfig,
	topologyExpire int,
) error {
	useTLS := (config.TLS != nil)
	timeout := logstashDefaultTimeout
	if config.Timeout != 0 {
		timeout = time.Duration(config.Timeout) * time.Second
	}

	defaultPort := logstashDefaultPort
	if config.Port != 0 {
		defaultPort = config.Port
	}

	maxWindowSize := defaultMaxWindowSize
	if config.BulkMaxSize != nil {
		maxWindowSize = *config.BulkMaxSize
	}

	compressLevel := defaultCompressionLevel
	if config.CompressionLevel != nil {
		compressLevel = *config.CompressionLevel
	}

	sendRetries := defaultSendRetries
	if config.MaxRetries != nil {
		sendRetries = *config.MaxRetries
	}
	logp.Info("Max Retries set to: %v", sendRetries)

	maxAttempts := sendRetries + 1
	if sendRetries < 0 {
		maxAttempts = 0
	}

	pipelining := false
	if config.Pipelined != nil {
		pipelining = *config.Pipelined
	}

	loadBalance := false
	if config.LoadBalance != nil {
		loadBalance = *config.LoadBalance
	} else if len(config.Hosts) <= 1 {
		loadBalance = config.Worker > 1
	}

	var makeTransp func(string) (TransportClient, error)
	if useTLS {
		tlsConfig, err := outputs.LoadTLSConfig(config.TLS)
		if err != nil {
			return err
		}
		makeTransp = makeTLSClient(defaultPort, tlsConfig)
	} else {
		makeTransp = makeTCPClient(defaultPort)
	}

	var m mode.ConnectionMode
	if pipelining {
		logp.Info("load balanced pipelining mode")

		clients, err := mode.MakeAsyncClients(config, makeAsyncClientFactory(
			maxWindowSize, compressLevel, timeout, makeTransp))
		if err != nil {
			return err
		}

		m, err = mode.NewAsyncConnectionMode(clients, !loadBalance,
			maxAttempts, waitRetry, timeout, maxWaitRetry)
		if err != nil {
			return err
		}
	} else {
		logp.Info("disabled pipelining")

		clients, err := mode.MakeClients(config,
			makeClientFactory(maxWindowSize, compressLevel, timeout, makeTransp))
		if err != nil {
			return err
		}

		m, err = mode.NewConnectionMode(clients, !loadBalance,
			maxAttempts, waitRetry, timeout, maxWaitRetry)
		if err != nil {
			return err
		}
	}

	lj.mode = m
	lj.index = config.Index

	return nil
}

func makeClientFactory(
	maxWindowSize int,
	compressLevel int,
	timeout time.Duration,
	makeTransp func(string) (TransportClient, error),
) func(string) (mode.ProtocolClient, error) {
	return func(host string) (mode.ProtocolClient, error) {
		transp, err := makeTransp(host)
		if err != nil {
			return nil, err
		}
		return newLumberjackClient(transp, compressLevel, maxWindowSize, timeout)
	}
}

func makeAsyncClientFactory(
	maxWindowSize int,
	compressLevel int,
	timeout time.Duration,
	makeTransp func(string) (TransportClient, error),
) func(string) (mode.AsyncProtocolClient, error) {
	return func(host string) (mode.AsyncProtocolClient, error) {
		trans, err := makeTransp(host)
		if err != nil {
			return nil, err
		}
		return newAsyncLumberjackClient(trans, compressLevel, maxWindowSize, timeout)
	}
}

func makeTCPClient(port int) func(string) (TransportClient, error) {
	return func(host string) (TransportClient, error) {
		return newTCPClient(host, port)
	}
}

func makeTLSClient(port int, tls *tls.Config) func(string) (TransportClient, error) {
	return func(host string) (TransportClient, error) {
		return newTLSClient(host, port, tls)
	}
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
