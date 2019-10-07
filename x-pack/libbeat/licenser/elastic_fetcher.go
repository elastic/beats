// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package licenser

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs/elasticsearch"
)

const xPackURL = "/_xpack"

// params defaults query parameters to send to the '_xpack' endpoint by default we only need
// machine parseable data.
var params = map[string]string{
	"human": "false",
}

var stateLookup = map[string]State{
	"inactive": Inactive,
	"active":   Active,
}

var licenseLookup = map[string]LicenseType{
	"oss":      OSS,
	"trial":    Trial,
	"standard": Standard,
	"basic":    Basic,
	"gold":     Gold,
	"platinum": Platinum,
}

// UnmarshalJSON takes a bytes array and convert it to the appropriate license type.
func (t *LicenseType) UnmarshalJSON(b []byte) error {
	if len(b) <= 2 {
		return fmt.Errorf("invalid string for license type, received: '%s'", string(b))
	}
	s := string(b[1 : len(b)-1])
	if license, ok := licenseLookup[s]; ok {
		*t = license
		return nil
	}

	return fmt.Errorf("unknown license type, received: '%s'", s)
}

// UnmarshalJSON takes a bytes array and convert it to the appropriate state.
func (st *State) UnmarshalJSON(b []byte) error {
	// we are only interested in the content between the quotes.
	if len(b) <= 2 {
		return fmt.Errorf("invalid string for state, received: '%s'", string(b))
	}

	s := string(b[1 : len(b)-1])
	if state, ok := stateLookup[s]; ok {
		*st = state
		return nil
	}
	return fmt.Errorf("unknown state, received: '%s'", s)
}

// UnmarshalJSON takes a bytes array and transform the int64 to a golang time.
func (et *expiryTime) UnmarshalJSON(b []byte) error {
	ts, err := strconv.ParseInt(string(b), 0, 64)
	if err != nil {
		return errors.Wrap(err, "could not parse value for expiry time")
	}

	*et = expiryTime(time.Unix(0, int64(time.Millisecond)*int64(ts)).UTC())
	return nil
}

type esclient interface {
	Request(
		method,
		path string,
		pipeline string,
		params map[string]string,
		body interface{},
	) (int, []byte, error)
}

// ElasticFetcher wraps an elasticsearch clients to retrieve licensing information
// on a specific cluster.
type ElasticFetcher struct {
	client esclient
	log    *logp.Logger
}

// NewElasticFetcher creates a new Elastic Fetcher
func NewElasticFetcher(client esclient) *ElasticFetcher {
	return &ElasticFetcher{client: client, log: logp.NewLogger("elasticfetcher")}
}

// Fetch retrieves the license information from an Elasticsearch Client, it will call the `_xpack`
// end point and will return a parsed license. If the `_xpack` endpoint is unreacheable we will
// return the OSS License otherwise we return an error.
func (f *ElasticFetcher) Fetch() (*License, error) {
	status, body, err := f.client.Request("GET", xPackURL, "", params, nil)
	// When we are running an OSS release of elasticsearch the _xpack endpoint will return a 405,
	// "Method Not Allowed", so we return the default OSS license.
	if status == http.StatusBadRequest {
		f.log.Debug("Received 'Bad request' (400) response from server, fallback to OSS license")
		return OSSLicense, nil
	}

	if status == http.StatusMethodNotAllowed {
		f.log.Debug("Received 'Method Not allowed' (405) response from server, fallback to OSS license")
		return OSSLicense, nil
	}

	if status == http.StatusUnauthorized {
		return nil, errors.New("unauthorized access, could not connect to the xpack endpoint, verify your credentials")
	}

	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve the license information from the cluster")
	}

	if status != http.StatusOK {
		return nil, fmt.Errorf("error from server, response code: %d", status)
	}

	license, err := f.parseJSON(body)
	if err != nil {
		f.log.Debugw("Invalid response from server", "body", string(body))
		return nil, errors.Wrap(err, "could not extract license information from the server response")
	}

	return license, nil
}

// Xpack Response, temporary struct to merge the features into the license struct.
type xpackResponse struct {
	License  License  `json:"license"`
	Features features `json:"features"`
}

func (f *ElasticFetcher) parseJSON(b []byte) (*License, error) {
	info := &xpackResponse{}

	if err := json.Unmarshal(b, info); err != nil {
		return nil, err
	}

	license := info.License
	license.Features = info.Features

	return &license, nil
}

// esClientMux is taking care of round robin request over an array of elasticsearch client, note that
// calling request is not threadsafe.
type esClientMux struct {
	clients []elasticsearch.Client
	idx     int
}

// Request takes a slice of elasticsearch clients and connect to one randomly and close the connection
// at the end of the function call, if an error occur we return the error and will pick up the next client on the
// next call. Not that we just round robin between hosts, any backoff strategy should be handled by
// the consumer of this type.
func (mux *esClientMux) Request(
	method, path string,
	pipeline string,
	params map[string]string,
	body interface{},
) (int, []byte, error) {
	c := mux.clients[mux.idx]

	if err := c.Connect(); err != nil {
		return 0, nil, err
	}
	defer c.Close()

	status, response, err := c.Request(method, path, pipeline, params, body)
	if err != nil {
		// use next host for next retry
		mux.idx = (mux.idx + 1) % len(mux.clients)
	}
	return status, response, err
}

// newESClientMux takes a list of clients and randomize where we start and the list of  host we are
// querying.
func newESClientMux(clients []elasticsearch.Client) *esClientMux {
	// randomize where we start
	idx := rand.Intn(len(clients))

	// randomize the list of round robin hosts.
	tmp := make([]elasticsearch.Client, len(clients))
	copy(tmp, clients)
	rand.Shuffle(len(tmp), func(i, j int) {
		tmp[i], tmp[j] = tmp[j], tmp[i]
	})

	return &esClientMux{idx: idx, clients: tmp}
}

// Create takes a raw configuration and will create a a license manager based on the elasticsearch
// output configuration, if no output is found we return an error.
func Create(cfg *common.ConfigNamespace, refreshDelay, graceDelay time.Duration) (*Manager, error) {
	if !cfg.IsSet() || cfg.Name() != "elasticsearch" {
		return nil, ErrNoElasticsearchConfig
	}

	clients, err := elasticsearch.NewElasticsearchClients(cfg.Config())
	if err != nil {
		return nil, err
	}
	clientsMux := newESClientMux(clients)

	manager := New(clientsMux, refreshDelay, graceDelay)
	return manager, nil
}
