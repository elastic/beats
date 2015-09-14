package lumberjack

// lumberjack.go defines the lumberjack plugin as being registered with all
// output plugins

import (
	"errors"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/outputs"
)

func init() {
	outputs.RegisterOutputPlugin("lumberjack", lumberjackOutputPlugin{})
}

type lumberjackOutputPlugin struct{}

func (p lumberjackOutputPlugin) NewOutput(
	beat string,
	config outputs.MothershipConfig,
	topologyExpire int,
) (outputs.Outputer, error) {
	output := &lumberjack{}
	err := output.init(beat, config, topologyExpire)
	if err != nil {
		return nil, err
	}
	return output, nil
}

type lumberjack struct {
	mode ConnectionMode
}

const lumberjackDefaultTimeout = 5 * time.Second

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
		useTLS = *config.TLS
	}

	timeout := lumberjackDefaultTimeout
	if config.Timeout != 0 {
		timeout = time.Duration(config.Timeout) * time.Second
	}

	var clients []ProtocolClient
	var err error
	if useTLS {
		var tlsConfig *tlsConfig
		tlsConfig, err = loadTLSConfig(&TLSConfig{
			Certificate: config.Certificate,
			Key:         config.CertificateKey,
			CAs:         config.CAs})
		if err != nil {
			return err
		}

		clients, err = makeClients(config, timeout,
			func(host string) (TransportClient, error) {
				return newTLSClient(host, *tlsConfig)
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

	var mode ConnectionMode
	if len(clients) == 1 {
		mode, err = newSingleConnectionMode(clients[0], waitRetry, timeout)
	} else {
		mode, err = newFailOverConnectionMode(clients, waitRetry, timeout)
	}
	if err != nil {
		return err
	}

	lj.mode = mode
	return nil
}

func makeClients(
	config outputs.MothershipConfig,
	timeout time.Duration,
	newTransp func(string) (TransportClient, error),
) ([]ProtocolClient, error) {
	switch {
	case len(config.Hosts) > 0:
		var clients []ProtocolClient
		for _, host := range config.Hosts {
			transp, err := newTransp(host)
			if err != nil {
				for _, client := range clients {
					client.Close()
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
		return []ProtocolClient{newLumberjackClient(transp, timeout)}, nil
	default:
		return nil, ErrNoHostsConfigured
	}
}

// TODO: update Outputer interface to support multiple events for batch-like
//       processing (e.g. for filebeat). Batch like processing might reduce
//       send/receive overhead per event for other implementors too.
func (lj *lumberjack) PublishEvent(
	trans outputs.Signaler,
	ts time.Time,
	event common.MapStr,
) error {
	events := []common.MapStr{event}
	return lj.mode.PublishEvents(trans, events)
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
