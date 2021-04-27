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
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs/elasticsearch"
)

// Enforce setups the corresponding callbacks in libbeat to verify the license on the
// remote elasticsearch cluster.
func Enforce(name string, checks ...CheckFunc) {
	name = strings.Title(name)

	cb := func(client *eslegclient.Connection) error {
		// Logger created earlier than this place are at risk of discarding any log statement.
		log := logp.NewLogger("elasticsearch")

		fetcher := NewElasticFetcher(client)
		license, err := fetcher.Fetch()

		if err != nil {
			return errors.Wrapf(err, "cannot retrieve the elasticsearch license from the /_license endpoint, "+
				"%s requires the default distribution of Elasticsearch.", name)
		}

		if !Validate(log, *license, checks...) {
			return fmt.Errorf(
				"invalid license found, requires a basic or a valid trial license and received %s",
				license.Get(),
			)
		}

		log.Infof("Elasticsearch license: %s", license.Get())

		return nil
	}

	elasticsearch.RegisterGlobalCallback(cb)
}
