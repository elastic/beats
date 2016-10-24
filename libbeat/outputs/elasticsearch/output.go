package elasticsearch

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/mode"
	"github.com/elastic/beats/libbeat/outputs/mode/modeutil"
	"github.com/elastic/beats/libbeat/outputs/outil"
	"github.com/elastic/beats/libbeat/outputs/transport"
	"github.com/elastic/beats/libbeat/paths"
)

type elasticsearchOutput struct {
	index    outil.Selector
	beatName string
	pipeline *outil.Selector

	mode mode.ConnectionMode
	topology

	template      map[string]interface{}
	template2x    map[string]interface{}
	templateMutex sync.Mutex
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

// New instantiates a new output plugin instance publishing to elasticsearch.
func New(beatName string, cfg *common.Config, topologyExpire int) (outputs.Outputer, error) {
	if !cfg.HasField("bulk_max_size") {
		cfg.SetInt("bulk_max_size", -1, defaultBulkSize)
	}

	if !cfg.HasField("index") {
		pattern := fmt.Sprintf("%v-%%{+yyyy.MM.dd}", beatName)
		cfg.SetString("index", -1, pattern)
	}

	output := &elasticsearchOutput{beatName: beatName}
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

	err = out.readTemplate(&config.Template)
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

// readTemplates reads the ES mapping template from the disk, if configured.
func (out *elasticsearchOutput) readTemplate(config *Template) error {
	if config.Enabled {
		// Set the defaults that depend on the beat name
		if config.Name == "" {
			config.Name = out.beatName
		}
		if config.Path == "" {
			config.Path = fmt.Sprintf("%s.template.json", out.beatName)
		}
		if config.Versions.Es2x.Path == "" {
			config.Versions.Es2x.Path = fmt.Sprintf("%s.template-es2x.json", out.beatName)
		}

		// Look for the template in the configuration path, if it's not absolute
		templatePath := paths.Resolve(paths.Config, config.Path)
		logp.Info("Loading template enabled. Reading template file: %v", templatePath)

		template, err := readTemplate(templatePath)
		if err != nil {
			return fmt.Errorf("Error loading template %s: %v", templatePath, err)
		}
		out.template = template

		if config.Versions.Es2x.Enabled {
			// Read the version of the template compatible with ES 2.x
			templatePath := paths.Resolve(paths.Config, config.Versions.Es2x.Path)
			logp.Info("Loading template enabled for Elasticsearch 2.x. Reading template file: %v", templatePath)

			template, err := readTemplate(templatePath)
			if err != nil {
				return fmt.Errorf("Error loading template %s: %v", templatePath, err)
			}
			out.template2x = template
		}
	}
	return nil
}

func readTemplate(filename string) (map[string]interface{}, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var template map[string]interface{}
	dec := json.NewDecoder(f)
	err = dec.Decode(&template)
	if err != nil {
		return nil, err
	}

	return template, nil
}

// loadTemplate checks if the index mapping template should be loaded
// In case the template is not already loaded or overwriting is enabled, the
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

		template := out.template
		if config.Versions.Es2x.Enabled && strings.HasPrefix(client.Connection.version, "2.") {
			logp.Info("Detected Elasticsearch 2.x. Automatically selecting the 2.x version of the template")
			template = out.template2x
		}

		err := client.LoadTemplate(config.Name, template)
		if err != nil {
			return fmt.Errorf("Could not load template: %v", err)
		}
	} else {
		logp.Info("Template already exists and will not be overwritten.")
	}

	return nil
}

func makeClientFactory(
	tls *transport.TLSConfig,
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
		if out.template != nil {
			onConnected = func(client *Client) error {
				return out.loadTemplate(config.Template, client)
			}
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
			Timeout:          config.Timeout,
			CompressionLevel: config.CompressionLevel,
		}, onConnected)
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

func parseProxyURL(raw string) (*url.URL, error) {
	url, err := url.Parse(raw)
	if err == nil && strings.HasPrefix(url.Scheme, "http") {
		return url, err
	}

	// Proxy was bogus. Try prepending "http://" to it and
	// see if that parses correctly.
	return url.Parse("http://" + raw)
}
