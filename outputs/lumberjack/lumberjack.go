package lumberjack

// lumberjack.go defines the lumberjack plugin as being registered with all
// output plugins

import (
	"crypto/tls"
	"errors"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/outputs"
	"github.com/elastic/libbeat/outputs/mode"
)

var debug = logp.MakeDebug("lumberjack")

func init() {
	outputs.RegisterOutputPlugin("lumberjack", lumberjackOutputPlugin{})
}

type lumberjackOutputPlugin struct{}

func (p lumberjackOutputPlugin) NewOutput(
	beat string,
	config *outputs.MothershipConfig,
	topologyExpire int,
) (outputs.Outputer, error) {
	output := &lumberjack{}
	err := output.init(beat, *config, topologyExpire)
	if err != nil {
		return nil, err
	}
	return output, nil
}

type lumberjack struct {
	mode mode.ConnectionMode
}

const (
	lumberjackDefaultTimeout = 5 * time.Second
	defaultSendRetries       = 3
)

// ErrNoHostsConfigured indicates missing host or hosts configuration
var ErrNoHostsConfigured = errors.New("no host configuration found")

var waitRetry = time.Duration(1) * time.Second

func (lj *lumberjack) init(
	beat string,
	config outputs.MothershipConfig,
	topologyExpire int,
) error {
	useTLS := false
	if config.TLS != nil {
		useTLS = !config.TLS.Disabled
	}

	timeout := lumberjackDefaultTimeout
	if config.Timeout != 0 {
		timeout = time.Duration(config.Timeout) * time.Second
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
				return newTLSClient(host, tlsConfig)
			})
	} else {
		clients, err = makeClients(config, timeout,
			func(host string) (TransportClient, error) {
				return newTCPClient(host)
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
		loadBalance := config.LoadBalance == nil || *config.LoadBalance
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
func (lj *lumberjack) PublishEvent(
	signaler outputs.Signaler,
	ts time.Time,
	event common.MapStr,
) error {
	return lj.mode.PublishEvent(signaler, event)
}

// BulkPublish implements the BulkOutputer interface pushing a bulk of events
// via lumberjack.
func (lj *lumberjack) BulkPublish(
	trans outputs.Signaler,
	ts time.Time,
	events []common.MapStr,
) error {
	return lj.mode.PublishEvents(trans, events)
}
