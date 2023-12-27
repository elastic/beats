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
	"github.com/elastic/elastic-agent-libs/logp"
)

// Verify checks if the connection endpoint is really Elasticsearch.
func FetchAndVerify(client *eslegclient.Connection) error {
	// Logger created earlier than this place are at risk of discarding any log statement.
	log := logp.NewLogger("elasticsearch")

	fetcher := NewElasticFetcher(client)
	license, err := fetcher.Fetch()
	if err != nil {
		return fmt.Errorf("could not connect to a compatible version of Elasticsearch: %w", err)
	}

	// Only notify users if they have an Elasticsearch license that has been expired.
	// We still will continue publish events as usual.
	if IsExpired(license) {
		log.Warn("Elasticsearch license is not active, please check Elasticsearch's licensing information at https://www.elastic.co/subscriptions.")
	}

	return nil
}
