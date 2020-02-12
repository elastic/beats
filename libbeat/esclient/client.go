package client

import (
	"encoding/json"
	"errors"

	"github.com/elastic/beats/libbeat/common"

	"github.com/elastic/go-elasticsearch"
	es6 "github.com/elastic/go-elasticsearch/v6"
	es7 "github.com/elastic/go-elasticsearch/v7"
	es8 "github.com/elastic/go-elasticsearch/v8"
)

// Client provides a version-agnostic Elasticsearch client. It wraps various
// lower-level version-aware Elasticsearch clients, instantiating the right one
// for the Elasticsearch cluster being connected to.
type Client struct {
	e6 *es6.Client
	e7 *es7.Client
	e8 *es8.Client

	version *common.Version

	addr []string

	username string
	password string
	apiToken string
}

type Option func(c *Client) error

// WithAddresses allows you to specify addresses for the Elasticsearch
// nodes to connect to. Addresses are of the form "http://localhost:9200".
func WithAddresses(addr ...string) Option {
	return func(c *Client) error {
		c.addr = addr
		return nil
	}
}

// WithBasicAuth allows you to specify a username and password to use
// when authenticating with the Elasticsearch cluster.
func WithBasicAuth(username, password string) Option {
	return func(c *Client) error {
		c.username = username
		c.password = password
		return nil
	}
}

// WithAPITopken allows you to specify an API token to use
// when authenticating with the Elasticsearch cluster.
func WithAPIToken(apiToken string) Option {
	return func(c *Client) error {
		c.apiToken = apiToken
		return nil
	}
}

// Constructor
func New(opts ...Option) (*Client, error) {
	c := &Client{
		addr: []string{"http://localhost:9200"},
	}

	for _, opt := range opts {
		err := opt(c)
		if err != nil {
			return nil, err
		}
	}

	e, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: c.addr,
	})
	if err != nil {
		return nil, err
	}

	version, err := getClusterVersion(e)
	if err != nil {
		return nil, err
	}

	c.version = version

	switch c.version.Major {
	case 6:
		c.e6, err = es6.NewClient(es6.Config{
			Addresses: c.addr,
		})
	case 7:
		c.e7, err = es7.NewClient(es7.Config{
			Addresses: c.addr,
		})
	case 8:
		c.e8, err = es8.NewClient(es8.Config{
			Addresses: c.addr,
		})
	default:
		return nil, ErrUnsupportedVersion
	}

	if err != nil {
		return nil, err
	}

	return c, nil
}

func getClusterVersion(e *elasticsearch.Client) (*common.Version, error) {
	r, err := e.Info()
	if err != nil {
		return nil, err
	}

	if r.IsError() {
		return nil, errors.New(r.String())
	}

	var info struct {
		Version struct {
			Number *common.Version
		}
	}

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
		return nil, err
	}

	return info.Version.Number, nil
}
