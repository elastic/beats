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

	logstashDefaultTimeout   = 30 * time.Second
	logstasDefaultMaxTimeout = 90 * time.Second
	defaultSendRetries       = 3
	defaultMaxWindowSize     = 1024
	defaultCompressionLevel  = 3
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

	var clients []mode.ProtocolClient
	var err error
	if useTLS {
		var tlsConfig *tls.Config
		tlsConfig, err = outputs.LoadTLSConfig(config.TLS)
		if err != nil {
			return err
		}

		clients, err = mode.MakeClients(config,
			makeClientFactory(config.Index, maxWindowSize, compressLevel, timeout,
				makeTLSClient(defaultPort, tlsConfig)))
	} else {
		clients, err = mode.MakeClients(config,
			makeClientFactory(config.Index, maxWindowSize, compressLevel, timeout,
				makeTCPClient(defaultPort)))
	}
	if err != nil {
		return err
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

	var m mode.ConnectionMode
	if len(clients) == 1 {
		m, err = mode.NewSingleConnectionMode(clients[0],
			maxAttempts, waitRetry, timeout, maxWaitRetry)
	} else {
		loadBalance := config.LoadBalance != nil && *config.LoadBalance
		if loadBalance {
			m, err = mode.NewLoadBalancerMode(clients, maxAttempts,
				waitRetry, timeout, maxWaitRetry)
		} else {
			m, err = mode.NewFailOverConnectionMode(clients, maxAttempts, waitRetry, timeout)
		}
	}
	if err != nil {
		return err
	}

	lj.mode = m
	lj.index = config.Index

	return nil
}

func makeClientFactory(
	beat string,
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
		return newLumberjackClient(transp, compressLevel, maxWindowSize, timeout, beat)
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
	return lj.mode.PublishEvent(signaler, opts, event)
}

// BulkPublish implements the BulkOutputer interface pushing a bulk of events
// via lumberjack.
func (lj *logstash) BulkPublish(
	trans outputs.Signaler,
	opts outputs.Options,
	events []common.MapStr,
) error {
	return lj.mode.PublishEvents(trans, opts, events)
}
