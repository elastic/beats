// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package licenser

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/elastic-agent-libs/logp"
)

const licenseURL = "/_license"

// params defaults query parameters to send to the '_license' endpoint by default we only need
// machine parseable data.
var params = map[string]string{
	"human": "false",
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

// Fetch retrieves the license information from an Elasticsearch Client, it will call the `_license`
// endpoint and will return a parsed license. If the `_license` endpoint is unreacheable we will
// return the OSS License otherwise we return an error.
func (f *ElasticFetcher) Fetch() (License, error) {
	status, body, err := f.client.Request("GET", licenseURL, "", params, nil)
	if status == http.StatusUnauthorized {
		return License{}, errors.New("unauthorized access, could not connect to the xpack endpoint, verify your credentials")
	}
	if err != nil {
		return License{}, err
	}

	if status != http.StatusOK {
		return License{}, fmt.Errorf("error from server, response code: %d", status)
	}

	license, err := f.parseJSON(body)
	if err != nil {
		f.log.Debugw("Invalid response from server", "body", string(body))
		return License{}, fmt.Errorf("failed to parse /_license response: %w", err)
	}

	return license, nil
}

// Xpack Response, temporary struct to merge the features into the license struct.
type xpackResponse struct {
	License License `json:"license"`
}

func (f *ElasticFetcher) parseJSON(b []byte) (License, error) {
	var info xpackResponse
	if err := json.Unmarshal(b, &info); err != nil {
		return License{}, err
	}
	return info.License, nil
}

// esClientMux is taking care of round robin request over an array of elasticsearch client, note that
// calling request is not threadsafe.
type esClientMux struct {
	clients []eslegclient.Connection
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
func newESClientMux(clients []eslegclient.Connection) *esClientMux {
	// randomize where we start
	idx := rand.Intn(len(clients))

	// randomize the list of round robin hosts.
	tmp := make([]eslegclient.Connection, len(clients))
	copy(tmp, clients)
	rand.Shuffle(len(tmp), func(i, j int) {
		tmp[i], tmp[j] = tmp[j], tmp[i]
	})

	return &esClientMux{idx: idx, clients: tmp}
}
