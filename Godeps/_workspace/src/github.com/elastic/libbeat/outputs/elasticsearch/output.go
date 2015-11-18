package elasticsearch

import (
	"crypto/tls"
	"errors"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/outputs"
	"github.com/elastic/libbeat/outputs/mode"
)

var debug = logp.MakeDebug("elasticsearch")

var (
	// ErrNotConnected indicates failure due to client having no valid connection
	ErrNotConnected = errors.New("not connected")

	// ErrJSONEncodeFailed indicates encoding failures
	ErrJSONEncodeFailed = errors.New("json encode failed")

	// ErrResponseRead indicates error parsing Elasticsearch response
	ErrResponseRead = errors.New("bulk item status parse failed.")
)

const (
	defaultMaxRetries = 3

	defaultBulkSize = 50

	elasticsearchDefaultTimeout = 90 * time.Second
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

	// configure bulk size in config in case it is not set
	if config.BulkMaxSize == nil {
		bulkSize := defaultBulkSize
		config.BulkMaxSize = &bulkSize
	}

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
	tlsConfig, err := outputs.LoadTLSConfig(config.TLS)
	if err != nil {
		return err
	}

	clients, err := mode.MakeClients(config, makeClientFactory(beat, tlsConfig, config))
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
	maxAttempts := maxRetries + 1 // maximum number of send attempts (-1 = infinite)
	if maxRetries < 0 {
		maxAttempts = 0
	}

	var waitRetry = time.Duration(1) * time.Second
	var maxWaitRetry = time.Duration(60) * time.Second

	var m mode.ConnectionMode
	out.clients = clients
	if len(clients) == 1 {
		client := clients[0]
		m, err = mode.NewSingleConnectionMode(client, maxAttempts,
			waitRetry, timeout, maxWaitRetry)
	} else {
		loadBalance := config.LoadBalance == nil || *config.LoadBalance
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

func makeClientFactory(
	beat string,
	tls *tls.Config,
	config outputs.MothershipConfig,
) func(string) (mode.ProtocolClient, error) {
	return func(host string) (mode.ProtocolClient, error) {
		url, err := getURL(config.Protocol, config.Path, host)
		if err != nil {
			logp.Err("Invalid host param set: %s, Error: %v", host, err)
			return nil, err
		}

		index := beat
		if config.Index != "" {
			index = config.Index
		}

		client := NewClient(url, index, tls, config.Username, config.Password)
		return client, nil
	}
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
