package elasticsearch

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/mode"
	"github.com/elastic/beats/libbeat/outputs/mode/modeutil"
	"github.com/elastic/beats/libbeat/paths"
)

type elasticsearchOutput struct {
	index string
	mode  mode.ConnectionMode
	topology

	templateContents []byte
	templateMutex    sync.Mutex
}

func init() {
	outputs.RegisterOutputPlugin("elasticsearch", New)
}

var (
	debug = logp.MakeDebug("elasticsearch")
)

var (
	// ErrNotConnected indicates failure due to client having no valid connection
	ErrNotConnected = errors.New("not connected")

	// ErrJSONEncodeFailed indicates encoding failures
	ErrJSONEncodeFailed = errors.New("json encode failed")

	// ErrResponseRead indicates error parsing Elasticsearch response
	ErrResponseRead = errors.New("bulk item status parse failed")
)

// New instantiates a new output plugin instance publishing to elasticsearch.
func New(cfg *common.Config, topologyExpire int) (outputs.Outputer, error) {
	if !cfg.HasField("bulk_max_size") {
		cfg.SetInt("bulk_max_size", -1, defaultBulkSize)
	}

	output := &elasticsearchOutput{}
	err := output.init(cfg, topologyExpire)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func (out *elasticsearchOutput) init(
	cfg *common.Config,
	topologyExpire int,
) error {
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return err
	}

	tlsConfig, err := outputs.LoadTLSConfig(config.TLS)
	if err != nil {
		return err
	}

	err = out.readTemplate(config.Template)
	if err != nil {
		return err
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
	m, err := modeutil.NewConnectionMode(clients, !loadBalance,
		maxAttempts, waitRetry, config.Timeout, maxWaitRetry)
	if err != nil {
		return err
	}

	if config.SaveTopology {
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
	out.index = config.Index

	return nil
}

// readTemplates reads the ES mapping template from the disk, if configured.
func (out *elasticsearchOutput) readTemplate(config Template) error {
	if len(config.Name) > 0 {
		// Look for the template in the configuration path, if it's not absolute
		templatePath := paths.Resolve(paths.Config, config.Path)

		logp.Info("Loading template enabled. Reading template file: %v", templatePath)

		var err error
		out.templateContents, err = ioutil.ReadFile(templatePath)
		if err != nil {
			return fmt.Errorf("Error loading template %s: %v", templatePath, err)
		}
	}
	return nil
}

// loadTemplate checks if the index mapping template should be loaded
// In case the template is not already loaded or overwritting is enabled, the
// template is written to index
func (out *elasticsearchOutput) loadTemplate(config Template, client *Client) error {
	out.templateMutex.Lock()
	defer out.templateMutex.Unlock()

	logp.Info("Trying to load template for client: %s", client.Connection.URL)

	// Check if template already exist or should be overwritten
	exists := client.CheckTemplate(config.Name)
	if !exists || config.Overwrite {

		if config.Overwrite {
			logp.Info("Existing template will be overwritten, as overwrite is enabled.")
		}

		reader := bytes.NewReader(out.templateContents)
		err := client.LoadTemplate(config.Name, reader)
		if err != nil {
			return fmt.Errorf("Could not load template: %v", err)
		}
	} else {
		logp.Info("Template already exists and will not be overwritten.")
	}

	return nil
}

func makeClientFactory(
	tls *tls.Config,
	config *elasticsearchConfig,
	out *elasticsearchOutput,
) func(string) (mode.ProtocolClient, error) {
	return func(host string) (mode.ProtocolClient, error) {
		esURL, err := getURL(config.Protocol, config.Path, host)
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

		// define a callback to be called on connection
		var onConnected connectCallback
		if len(out.templateContents) > 0 {
			onConnected = func(client *Client) error {
				return out.loadTemplate(config.Template, client)
			}
		}

		client := NewClient(
			esURL, config.Index, proxyURL, tls,
			config.Username, config.Password,
			params, onConnected)
		return client, nil
	}
}

func (out *elasticsearchOutput) Close() error {
	return out.mode.Close()
}

func (out *elasticsearchOutput) PublishEvent(
	signaler op.Signaler,
	opts outputs.Options,
	event common.MapStr,
) error {
	return out.mode.PublishEvent(signaler, opts, event)
}

func (out *elasticsearchOutput) BulkPublish(
	trans op.Signaler,
	opts outputs.Options,
	events []common.MapStr,
) error {
	return out.mode.PublishEvents(trans, opts, events)
}

func parseProxyURL(raw string) (*url.URL, error) {
	url, err := url.Parse(raw)
	if err == nil && strings.HasPrefix(url.Scheme, "http") {
		return url, err
	}

	// Proxy was bogus. Try prepending "http://" to it and
	// see if that parses correctly.
	return url.Parse("http://" + raw)
}
