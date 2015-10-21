package logstash

// logstash.go defines the logtash plugin (using lumberjack protocol) as being registered with all
// output plugins

import (
	"crypto/tls"
	"errors"
	"fmt"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/outputs"
	"github.com/elastic/libbeat/outputs/mode"
)

var debug = logp.MakeDebug("logstash")

func init() {
	outputs.RegisterOutputPlugin("logstash", logstashOutputPlugin{})
}

type logstashOutputPlugin struct{}

func (p logstashOutputPlugin) NewOutput(
	beat string,
	config *outputs.MothershipConfig,
	topologyExpire int,
) (outputs.Outputer, error) {
	output := &logstash{}
	err := output.init(beat, *config, topologyExpire)
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

	logstashDefaultTimeout = 30 * time.Second
	defaultSendRetries     = 3
)

// ErrNoHostsConfigured indicates missing host or hosts configuration
var ErrNoHostsConfigured = errors.New("no host configuration found")

var waitRetry = time.Duration(1) * time.Second

func (lj *logstash) init(
	beat string,
	config outputs.MothershipConfig,
	topologyExpire int,
) error {
	useTLS := false
	if config.TLS != nil {
		useTLS = !config.TLS.Disabled
	}

	timeout := logstashDefaultTimeout
	if config.Timeout != 0 {
		timeout = time.Duration(config.Timeout) * time.Second
	}

	defaultPort := logstashDefaultPort
	if config.Port != 0 {
		defaultPort = config.Port
	}

	var clients []mode.ProtocolClient
	var err error
	if useTLS {
		var tlsConfig *tls.Config
		tlsConfig, err = outputs.LoadTLSConfig(config.TLS)
		if err != nil {
			return err
		}

		clients, err = makeClients(config, timeout,
			func(host string) (TransportClient, error) {
				return newTLSClient(host, defaultPort, tlsConfig)
			})
	} else {
		clients, err = makeClients(config, timeout,
			func(host string) (TransportClient, error) {
				return newTCPClient(host, defaultPort)
			})
	}
	if err != nil {
		return err
	}

	sendRetries := defaultSendRetries
	if config.Max_retries != nil {
		sendRetries = *config.Max_retries
	}

	var m mode.ConnectionMode
	if len(clients) == 1 {
		m, err = mode.NewSingleConnectionMode(clients[0],
			sendRetries, waitRetry, timeout)
	} else {
		loadBalance := config.LoadBalance != nil && *config.LoadBalance
		if loadBalance {
			m, err = mode.NewLoadBalancerMode(clients, sendRetries, waitRetry, timeout)
		} else {
			m, err = mode.NewFailOverConnectionMode(clients, sendRetries, waitRetry, timeout)
		}
	}
	if err != nil {
		return err
	}

	lj.mode = m
	if config.Index != "" {
		lj.index = config.Index
	} else {
		lj.index = beat
	}
	return nil
}

func makeClients(
	config outputs.MothershipConfig,
	timeout time.Duration,
	newTransp func(string) (TransportClient, error),
) ([]mode.ProtocolClient, error) {
	switch {
	case len(config.Hosts) > 0:
		var clients []mode.ProtocolClient
		for _, host := range config.Hosts {
			transp, err := newTransp(host)
			if err != nil {
				for _, client := range clients {
					_ = client.Close() // ignore error
				}
				return nil, err
			}
			client := newLumberjackClient(transp, timeout)
			clients = append(clients, client)
		}
		return clients, nil
	case config.Host != "":
		transp, err := newTransp(config.Host)
		if err != nil {
			return nil, err
		}
		return []mode.ProtocolClient{newLumberjackClient(transp, timeout)}, nil
	default:
		return nil, ErrNoHostsConfigured
	}
}

// TODO: update Outputer interface to support multiple events for batch-like
//       processing (e.g. for filebeat). Batch like processing might reduce
//       send/receive overhead per event for other implementors too.
func (lj *logstash) PublishEvent(
	signaler outputs.Signaler,
	ts time.Time,
	event common.MapStr,
) error {
	lj.addMeta(event)
	return lj.mode.PublishEvent(signaler, event)
}

// BulkPublish implements the BulkOutputer interface pushing a bulk of events
// via lumberjack.
func (lj *logstash) BulkPublish(
	trans outputs.Signaler,
	ts time.Time,
	events []common.MapStr,
) error {
	for _, event := range events {
		lj.addMeta(event)
	}
	return lj.mode.PublishEvents(trans, events)
}

// addMeta adapts events to be compatible with logstash forwarer messages by renaming
// the "message" field to "line". The lumberjack server in logstash will
// decode/rename the "line" field into "message".
func (lj *logstash) addMeta(event common.MapStr) {
	// add metadata for indexing
	ts := time.Time(event["timestamp"].(common.Time)).UTC()
	index := fmt.Sprintf("%s-%02d.%02d.%02d", lj.index,
		ts.Year(), ts.Month(), ts.Day())
	event["@metadata"] = common.MapStr{
		"index": index,
		"type":  event["type"].(string),
	}
}
