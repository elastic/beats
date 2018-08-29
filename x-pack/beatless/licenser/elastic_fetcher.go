// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package licenser

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/pkg/errors"

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
	if len(b) < 0 {
		return fmt.Errorf("invalid value for expiry time, received: '%s'", string(b))
	}

	ts, err := strconv.Atoi(string(b))
	if err != nil {
		return errors.Wrap(err, "could not parse value for expiry time")
	}

	*et = expiryTime(time.Unix(0, int64(time.Millisecond)*int64(ts)).UTC())
	return nil
}

// ElasticFetcher wraps an elasticsearch clients to retrieve licensing information
// on a specific cluster.
type ElasticFetcher struct {
	client *elasticsearch.Client
	log    *logp.Logger
}

// NewElasticFetcher creates a new Elastic Fetcher
func NewElasticFetcher(client *elasticsearch.Client) *ElasticFetcher {
	return &ElasticFetcher{client: client, log: logp.NewLogger("elasticfetcher")}
}

// Fetch retrieves the license information from an Elasticsearch Client, it will call the `_xpack`
// end point and will return a parsed license. If the `_xpack` endpoint is unreacheable we will
// return the OSS License otherwise we return an error.
func (f *ElasticFetcher) Fetch() (*License, error) {
	status, body, err := f.client.Request("GET", xPackURL, "", params, nil)
	// When we are running an OSS release of elasticsearch the _xpack endpoint will return a 405,
	// "Method Not Allowed", so we return the default OSS license.
	if status == http.StatusMethodNotAllowed {
		f.log.Debug("received 'Method Not allowed' (405) response from server, fallback to OSS license")
		return OSSLicense, nil
	}

	if status == http.StatusUnauthorized {
		return nil, errors.New("Unauthorized access, could not connect to the xpack endpoint, verify your credentials")
	}

	if status != http.StatusOK {
		return nil, fmt.Errorf("could not retrieve license information, response code: %d", status)
	}

	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve the license information from the cluster")
	}

	license, err := f.parseJSON(body)
	if err != nil {
		f.log.Debugw("invalid response from server", "body", string(body))
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
