// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package elasticsearch

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/elastic/go-elasticsearch/v8/estransport"
	"github.com/elastic/go-elasticsearch/v8/internal/version"
)

const (
	defaultURL = "http://localhost:9200"
)

// Version returns the package version as a string.
//
const Version = version.Client

// Config represents the client configuration.
//
type Config struct {
	Addresses []string // A list of Elasticsearch nodes to use.
	Username  string   // Username for HTTP Basic Authentication.
	Password  string   // Password for HTTP Basic Authentication.

	CloudID string // Endpoint for the Elastic Service (https://elastic.co/cloud).
	APIKey  string // Base64-encoded token for authorization; if set, overrides username and password.

	// PEM-encoded certificate authorities.
	// When set, an empty certificate pool will be created, and the certificates will be appended to it.
	// The option is only valid when the transport is not specified, or when it's http.Transport.
	CACert []byte

	RetryOnStatus        []int // List of status codes for retry. Default: 502, 503, 504.
	DisableRetry         bool  // Default: false.
	EnableRetryOnTimeout bool  // Default: false.
	MaxRetries           int   // Default: 3.

	DiscoverNodesOnStart  bool          // Discover nodes when initializing the client. Default: false.
	DiscoverNodesInterval time.Duration // Discover nodes periodically. Default: disabled.

	EnableMetrics     bool // Enable the metrics collection.
	EnableDebugLogger bool // Enable the debug logging.

	RetryBackoff func(attempt int) time.Duration // Optional backoff duration. Default: nil.

	Transport http.RoundTripper    // The HTTP transport object.
	Logger    estransport.Logger   // The logger object.
	Selector  estransport.Selector // The selector object.

	// Optional constructor function for a custom ConnectionPool. Default: nil.
	ConnectionPoolFunc func([]*estransport.Connection, estransport.Selector) estransport.ConnectionPool
}

// Client represents the Elasticsearch client.
//
type Client struct {
	*esapi.API // Embeds the API methods
	Transport  estransport.Interface
}

// NewDefaultClient creates a new client with default options.
//
// It will use http://localhost:9200 as the default address.
//
// It will use the ELASTICSEARCH_URL environment variable, if set,
// to configure the addresses; use a comma to separate multiple URLs.
//
func NewDefaultClient() (*Client, error) {
	return NewClient(Config{})
}

// NewClient creates a new client with configuration from cfg.
//
// It will use http://localhost:9200 as the default address.
//
// It will use the ELASTICSEARCH_URL environment variable, if set,
// to configure the addresses; use a comma to separate multiple URLs.
//
// If either cfg.Addresses or cfg.CloudID is set, the ELASTICSEARCH_URL
// environment variable is ignored.
//
// It's an error to set both cfg.Addresses and cfg.CloudID.
//
func NewClient(cfg Config) (*Client, error) {
	var addrs []string

	if len(cfg.Addresses) == 0 && cfg.CloudID == "" {
		addrs = addrsFromEnvironment()
	} else {
		if len(cfg.Addresses) > 0 && cfg.CloudID != "" {
			return nil, errors.New("cannot create client: both Addresses and CloudID are set")
		}

		if cfg.CloudID != "" {
			cloudAddr, err := addrFromCloudID(cfg.CloudID)
			if err != nil {
				return nil, fmt.Errorf("cannot create client: cannot parse CloudID: %s", err)
			}
			addrs = append(addrs, cloudAddr)
		}

		if len(cfg.Addresses) > 0 {
			addrs = append(addrs, cfg.Addresses...)
		}
	}

	urls, err := addrsToURLs(addrs)
	if err != nil {
		return nil, fmt.Errorf("cannot create client: %s", err)
	}

	if len(urls) == 0 {
		u, _ := url.Parse(defaultURL) // errcheck exclude
		urls = append(urls, u)
	}

	// TODO(karmi): Refactor
	if urls[0].User != nil {
		cfg.Username = urls[0].User.Username()
		pw, _ := urls[0].User.Password()
		cfg.Password = pw
	}

	tp, err := estransport.New(estransport.Config{
		URLs:     urls,
		Username: cfg.Username,
		Password: cfg.Password,
		APIKey:   cfg.APIKey,

		CACert: cfg.CACert,

		RetryOnStatus:        cfg.RetryOnStatus,
		DisableRetry:         cfg.DisableRetry,
		EnableRetryOnTimeout: cfg.EnableRetryOnTimeout,
		MaxRetries:           cfg.MaxRetries,
		RetryBackoff:         cfg.RetryBackoff,

		EnableMetrics:     cfg.EnableMetrics,
		EnableDebugLogger: cfg.EnableDebugLogger,

		DiscoverNodesInterval: cfg.DiscoverNodesInterval,

		Transport:          cfg.Transport,
		Logger:             cfg.Logger,
		Selector:           cfg.Selector,
		ConnectionPoolFunc: cfg.ConnectionPoolFunc,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating transport: %s", err)
	}

	client := &Client{Transport: tp, API: esapi.New(tp)}

	if cfg.DiscoverNodesOnStart {
		go client.DiscoverNodes()
	}

	return client, nil
}

// Perform delegates to Transport to execute a request and return a response.
//
func (c *Client) Perform(req *http.Request) (*http.Response, error) {
	return c.Transport.Perform(req)
}

// Metrics returns the client metrics.
//
func (c *Client) Metrics() (estransport.Metrics, error) {
	if mt, ok := c.Transport.(estransport.Measurable); ok {
		return mt.Metrics()
	}
	return estransport.Metrics{}, errors.New("transport is missing method Metrics()")
}

// DiscoverNodes reloads the client connections by fetching information from the cluster.
//
func (c *Client) DiscoverNodes() error {
	if dt, ok := c.Transport.(estransport.Discoverable); ok {
		return dt.DiscoverNodes()
	}
	return errors.New("transport is missing method DiscoverNodes()")
}

// addrsFromEnvironment returns a list of addresses by splitting
// the ELASTICSEARCH_URL environment variable with comma, or an empty list.
//
func addrsFromEnvironment() []string {
	var addrs []string

	if envURLs, ok := os.LookupEnv("ELASTICSEARCH_URL"); ok && envURLs != "" {
		list := strings.Split(envURLs, ",")
		for _, u := range list {
			addrs = append(addrs, strings.TrimSpace(u))
		}
	}

	return addrs
}

// addrsToURLs creates a list of url.URL structures from url list.
//
func addrsToURLs(addrs []string) ([]*url.URL, error) {
	var urls []*url.URL
	for _, addr := range addrs {
		u, err := url.Parse(strings.TrimRight(addr, "/"))
		if err != nil {
			return nil, fmt.Errorf("cannot parse url: %v", err)
		}

		urls = append(urls, u)
	}
	return urls, nil
}

// addrFromCloudID extracts the Elasticsearch URL from CloudID.
// See: https://www.elastic.co/guide/en/cloud/current/ec-cloud-id.html
//
func addrFromCloudID(input string) (string, error) {
	var scheme = "https://"

	values := strings.Split(input, ":")
	if len(values) != 2 {
		return "", fmt.Errorf("unexpected format: %q", input)
	}
	data, err := base64.StdEncoding.DecodeString(values[1])
	if err != nil {
		return "", err
	}
	parts := strings.Split(string(data), "$")

	if len(parts) < 2 {
		return "", fmt.Errorf("invalid encoded value: %s", parts)
	}

	return fmt.Sprintf("%s%s.%s", scheme, parts[1], parts[0]), nil
}
