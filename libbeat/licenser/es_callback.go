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
	"fmt"

	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
)

// fetchElasticsearchLicense fetches Elasticsearch license
func fetchElasticsearchLicense(client *eslegclient.Connection) (*License, error) {
	// Logger created earlier than this place are at risk of discarding any log statement.
	log := logp.NewLogger("elasticsearch")

	fetcher := NewElasticFetcher(client)
	license, err := fetcher.Fetch()
	if err != nil {
		return nil, fmt.Errorf("could not connect to a compatible version of Elasticsearch: %w", err)
	}
	// Only notify users if they have an Elasticsearch license that has been expired.
	// We still will continue publish events as usual.
	if IsExpired(license) {
		log.Warn("Elasticsearch license is not active, please check Elasticsearch's licensing information at https://www.elastic.co/subscriptions.")
	}
	return &license, nil
}

// Verify checks if the connection endpoint is really Elasticsearch.
func FetchAndVerify(client *eslegclient.Connection) error {
	license, err := fetchElasticsearchLicense(client)
	if err != nil {
		return err
	}

	if license.Type == OSS {
		return errors.New("could not connect to a compatible version of Elasticsearch: found OSS license")
	}

	return nil
}

// IsElasticsearch checks if the connection endpoint is Elasticsearch.
func IsElasticsearch(client *eslegclient.Connection) error {
	log := logp.NewLogger("elasticsearch")
	license, err := fetchElasticsearchLicense(client)
	if err != nil {
		return err
	}
	if license.Type == OSS {
		log.Warn("DEPRECATION WARNING: Connecting to an OSS distribution of Elasticsearch using the default " +
			"distribution of Beats will stop working in Beats 8.0.0. Please upgrade to the " +
			"default distribution of Elasticsearch or use the OSS distribution of Beats")
	}
	return nil
}
