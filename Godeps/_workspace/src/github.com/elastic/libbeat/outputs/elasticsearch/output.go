package elasticsearch

import (
	"errors"
	"fmt"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/outputs"
	"github.com/elastic/libbeat/outputs/mode"
)

var debug = logp.MakeDebug("elasticsearch")

var (
	// ErrNoHostsConfigured indicates missing host or hosts configuration
	ErrNoHostsConfigured = errors.New("no host configuration found")

	// ErrNotConnected indicates failure due to client having no valid connection
	ErrNotConnected = errors.New("not connected")

	// ErrJSONEncodeFailed indicates encoding failures
	ErrJSONEncodeFailed = errors.New("json encode failed")
)

const (
	defaultEsOpenTimeout = 3000 * time.Millisecond

	defaultMaxRetries = 3

	elasticsearchDefaultTimeout = 30 * time.Second
)

func init() {
	outputs.RegisterOutputPlugin("elasticsearch", elasticsearchOutputPlugin{})
}

type elasticsearchOutputPlugin struct{}

type elasticsearchOutput struct {
	index string
	mode  mode.ConnectionMode

	topology
}

// NewOutput instantiates a new output plugin instance publishing to elasticsearch.
func (f elasticsearchOutputPlugin) NewOutput(
	beat string,
	config *outputs.MothershipConfig,
	topologyExpire int,
) (outputs.Outputer, error) {
	output := &elasticsearchOutput{}
	err := output.init(beat, *config, topologyExpire)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func (out *elasticsearchOutput) init(
	beat string,
	config outputs.MothershipConfig,
	topologyExpire int,
) error {
	clients, err := makeClients(beat, config)
	if err != nil {
		return err
	}

	timeout := elasticsearchDefaultTimeout
	if config.Timeout != 0 {
		timeout = time.Duration(config.Timeout) * time.Second
	}

	maxRetries := defaultMaxRetries
	if config.Max_retries != nil {
		maxRetries = *config.Max_retries
	}

	var waitRetry = time.Duration(1) * time.Second

	var m mode.ConnectionMode
	out.clients = clients
	if len(clients) == 1 {
		client := clients[0]
		m, err = mode.NewSingleConnectionMode(client, maxRetries, waitRetry, timeout)
	} else {
		loadBalance := config.LoadBalance == nil || *config.LoadBalance
		if loadBalance {
			m, err = mode.NewLoadBalancerMode(clients, maxRetries, waitRetry, timeout)
		} else {
			m, err = mode.NewFailOverConnectionMode(clients, maxRetries, waitRetry, timeout)
		}
	}
	if err != nil {
		return err
	}

	if config.Save_topology {
		err := out.EnableTTL()
		if err != nil {
			logp.Err("Fail to set _ttl mapping: %s", err)
			// keep trying in the background
			go func() {
				for {
					err := out.EnableTTL()
					if err == nil {
						break
					}
					logp.Err("Fail to set _ttl mapping: %s", err)
					time.Sleep(5 * time.Second)
				}
			}()
		}
	}

	out.TopologyExpire = 15000
	if topologyExpire != 0 {
		out.TopologyExpire = topologyExpire * 1000 // millisec
	}

	out.mode = m
	if config.Index != "" {
		out.index = config.Index
	} else {
		out.index = beat
	}
	return nil
}

func makeClients(
	beat string,
	config outputs.MothershipConfig,
) ([]mode.ProtocolClient, error) {
	var urls []string
	if len(config.Hosts) > 0 {
		// use hosts setting
		for _, host := range config.Hosts {
			url, err := getURL(config.Protocol, config.Path, host)
			if err != nil {
				logp.Err("Invalid host param set: %s, Error: %v", host, err)
			}

			urls = append(urls, url)
		}
	} else {
		// usage of host and port is deprecated as it is replaced by hosts
		url := fmt.Sprintf("%s://%s:%d%s", config.Protocol, config.Host, config.Port, config.Path)
		urls = append(urls, url)
	}

	index := beat
	if config.Index != "" {
		index = config.Index
	}

	tlsConfig, err := outputs.LoadTLSConfig(config.TLS)
	if err != nil {
		return nil, err
	}

	var clients []mode.ProtocolClient
	for _, url := range urls {
		client := NewClient(url, index, tlsConfig, config.Username, config.Password)
		clients = append(clients, client)
	}
	return clients, nil
}

func (out *elasticsearchOutput) PublishEvent(
	signaler outputs.Signaler,
	ts time.Time,
	event common.MapStr,
) error {
	return out.mode.PublishEvent(signaler, event)
}

func (out *elasticsearchOutput) BulkPublish(
	trans outputs.Signaler,
	ts time.Time,
	events []common.MapStr,
) error {
	return out.mode.PublishEvents(trans, events)
}
