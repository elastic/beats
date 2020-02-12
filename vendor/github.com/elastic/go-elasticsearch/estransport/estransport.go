package estransport // import "github.com/elastic/go-elasticsearch/estransport"

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Interface defines the interface for HTTP client.
//
type Interface interface {
	Perform(*http.Request) (*http.Response, error)
}

// Config represents the configuration of HTTP client.
//
type Config struct {
	URLs      []*url.URL
	Transport http.RoundTripper
}

// Client represents the HTTP client.
//
type Client struct {
	urls      []*url.URL
	transport http.RoundTripper
	selector  Selector
}

// New creates new HTTP client.
//
// http.DefaultTransport will be used if no transport
// is passed in the configuration.
//
func New(cfg Config) *Client {
	if cfg.Transport == nil {
		cfg.Transport = http.DefaultTransport
	}

	return &Client{
		urls:      cfg.URLs,
		transport: cfg.Transport,
		selector:  NewRoundRobinSelector(cfg.URLs...),
	}
}

// Perform executes the request and returns a response or error.
//
func (c *Client) Perform(req *http.Request) (*http.Response, error) {
	u, err := c.getURL()
	if err != nil {
		return nil, fmt.Errorf("cannot get URL: %s", err)
	}

	c.setURL(u, req)
	c.setBasicAuth(u, req)

	// TODO(karmi): Wrap error
	return c.transport.RoundTrip(req)
}

// URLs returns a list of transport URLs.
//
func (c *Client) URLs() []*url.URL {
	return c.urls
}

func (c *Client) getURL() (*url.URL, error) {
	return c.selector.Select()
}

func (c *Client) setURL(u *url.URL, req *http.Request) *http.Request {
	req.URL.Scheme = u.Scheme
	req.URL.Host = u.Host

	if u.Path != "" {
		var b strings.Builder
		b.Grow(len(u.Path) + len(req.URL.Path))
		b.WriteString(u.Path)
		b.WriteString(req.URL.Path)
		req.URL.Path = b.String()
	}

	return req
}

func (c *Client) setBasicAuth(u *url.URL, req *http.Request) *http.Request {
	if u.User != nil {
		password, _ := u.User.Password()
		req.SetBasicAuth(u.User.Username(), password)
	}
	return req
}
