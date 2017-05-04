package elasticsearch

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/mode"
	"github.com/elastic/beats/libbeat/outputs/mode/modeutil"
	"github.com/elastic/beats/libbeat/outputs/outil"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

type elasticsearchOutput struct {
	index    outil.Selector
	beat     common.BeatInfo
	pipeline *outil.Selector
	clients  []mode.ProtocolClient

	mode mode.ConnectionMode
}

func init() {
	outputs.RegisterOutputPlugin("elasticsearch", New)
}

var (
	debugf = logp.MakeDebug("elasticsearch")
)

var (
	// ErrNotConnected indicates failure due to client having no valid connection
	ErrNotConnected = errors.New("not connected")

	// ErrJSONEncodeFailed indicates encoding failures
	ErrJSONEncodeFailed = errors.New("json encode failed")

	// ErrResponseRead indicates error parsing Elasticsearch response
	ErrResponseRead = errors.New("bulk item status parse failed")
)

var connectCallbackRegistry connectCallback

// RegisterConnectCallback registers a callback for the elasticsearch output
// The callback is called each time the client connects to elasticsearch.
func RegisterConnectCallback(callback connectCallback) {
	connectCallbackRegistry = callback
}

// New instantiates a new output plugin instance publishing to elasticsearch.
func New(beat common.BeatInfo, cfg *common.Config) (outputs.Outputer, error) {
	if !cfg.HasField("bulk_max_size") {
		cfg.SetInt("bulk_max_size", -1, defaultBulkSize)
	}

	if !cfg.HasField("index") {
		pattern := fmt.Sprintf("%v-%v-%%{+yyyy.MM.dd}", beat.Beat, beat.Version)
		cfg.SetString("index", -1, pattern)
	}

	output := &elasticsearchOutput{beat: beat}
	err := output.init(cfg)
	if err != nil {
		return nil, err
	}
	return output, nil
}

// NewConnectedClient creates a new Elasticsearch client based on the given config.
// It uses the NewElasticsearchClients to create a list of clients then returns
// the first from the list that successfully connects.
func NewConnectedClient(cfg *common.Config) (*Client, error) {
	clients, err := NewElasticsearchClients(cfg)
	if err != nil {
		return nil, err
	}

	for _, client := range clients {
		err = client.Connect(client.timeout)
		if err != nil {
			logp.Err("Error connecting to Elasticsearch: %s", client.Connection.URL)
			continue
		}
		return &client, nil
	}
	return nil, fmt.Errorf("Couldn't connect to any of the configured Elasticsearch hosts")
}

// NewElasticsearchClients returns a list of Elasticsearch clients based on the given
// configuration. It accepts the same configuration parameters as the output,
// except for the output specific configuration options (index, pipeline,
// template) .If multiple hosts are defined in the configuration, a client is returned
// for each of them.
func NewElasticsearchClients(cfg *common.Config) ([]Client, error) {

	hosts, err := modeutil.ReadHostList(cfg)
	if err != nil {
		return nil, err
	}

	config := defaultConfig
	if err = cfg.Unpack(&config); err != nil {
		return nil, err
	}

	tlsConfig, err := outputs.LoadTLSConfig(config.TLS)
	if err != nil {
		return nil, err
	}

	proxyURL, err := parseProxyURL(config.ProxyURL)
	if err != nil {
		return nil, err
	}
	if proxyURL != nil {
		logp.Info("Using proxy URL: %s", proxyURL)
	}

	params := config.Params
	if len(params) == 0 {
		params = nil
	}

	clients := []Client{}
	for _, host := range hosts {
		esURL, err := MakeURL(config.Protocol, config.Path, host)
		if err != nil {
			logp.Err("Invalid host param set: %s, Error: %v", host, err)
			return nil, err
		}

		client, err := NewClient(ClientSettings{
			URL:              esURL,
			Proxy:            proxyURL,
			TLS:              tlsConfig,
			Username:         config.Username,
			Password:         config.Password,
			Parameters:       params,
			Headers:          config.Headers,
			Timeout:          config.Timeout,
			CompressionLevel: config.CompressionLevel,
		}, nil)
		if err != nil {
			return clients, err
		}
		clients = append(clients, *client)
	}
	if len(clients) == 0 {
		return clients, fmt.Errorf("No hosts defined in the Elasticsearch output")
	}
	return clients, nil
}

func (out *elasticsearchOutput) init(
	cfg *common.Config,
) error {
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return err
	}

	index, err := outil.BuildSelectorFromConfig(cfg, outil.Settings{
		Key:              "index",
		MultiKey:         "indices",
		EnableSingleOnly: true,
		FailEmpty:        true,
	})
	if err != nil {
		return err
	}

	tlsConfig, err := outputs.LoadTLSConfig(config.TLS)
	if err != nil {
		return err
	}

	out.index = index
	pipeline, err := outil.BuildSelectorFromConfig(cfg, outil.Settings{
		Key:              "pipeline",
		MultiKey:         "pipelines",
		EnableSingleOnly: true,
		FailEmpty:        false,
	})
	if err != nil {
		return err
	}

	if !pipeline.IsEmpty() {
		out.pipeline = &pipeline
	}

	clients, err := modeutil.MakeClients(cfg, makeClientFactory(tlsConfig, &config, out))
	if err != nil {
		return err
	}

	maxRetries := config.MaxRetries
	maxAttempts := maxRetries + 1 // maximum number of send attempts (-1 = infinite)
	if maxRetries < 0 {
		maxAttempts = 0
	}

	var waitRetry = time.Duration(1) * time.Second
	var maxWaitRetry = time.Duration(60) * time.Second

	out.clients = clients
	loadBalance := config.LoadBalance
	m, err := modeutil.NewConnectionMode(clients, modeutil.Settings{
		Failover:     !loadBalance,
		MaxAttempts:  maxAttempts,
		Timeout:      config.Timeout,
		WaitRetry:    waitRetry,
		MaxWaitRetry: maxWaitRetry,
	})
	if err != nil {
		return err
	}

	out.mode = m

	return nil
}

func makeClientFactory(
	tls *transport.TLSConfig,
	config *elasticsearchConfig,
	out *elasticsearchOutput,
) func(string) (mode.ProtocolClient, error) {
	return func(host string) (mode.ProtocolClient, error) {
		esURL, err := MakeURL(config.Protocol, config.Path, host)
		if err != nil {
			logp.Err("Invalid host param set: %s, Error: %v", host, err)
			return nil, err
		}

		var proxyURL *url.URL
		if config.ProxyURL != "" {
			proxyURL, err = parseProxyURL(config.ProxyURL)
			if err != nil {
				return nil, err
			}

			logp.Info("Using proxy URL: %s", proxyURL)
		}

		params := config.Params
		if len(params) == 0 {
			params = nil
		}

		return NewClient(ClientSettings{
			URL:              esURL,
			Index:            out.index,
			Pipeline:         out.pipeline,
			Proxy:            proxyURL,
			TLS:              tls,
			Username:         config.Username,
			Password:         config.Password,
			Parameters:       params,
			Headers:          config.Headers,
			Timeout:          config.Timeout,
			CompressionLevel: config.CompressionLevel,
		}, connectCallbackRegistry)
	}
}

func (out *elasticsearchOutput) Close() error {
	return out.mode.Close()
}

func (out *elasticsearchOutput) PublishEvent(
	signaler op.Signaler,
	opts outputs.Options,
	data outputs.Data,
) error {
	return out.mode.PublishEvent(signaler, opts, data)
}

func (out *elasticsearchOutput) BulkPublish(
	trans op.Signaler,
	opts outputs.Options,
	data []outputs.Data,
) error {
	return out.mode.PublishEvents(trans, opts, data)
}
