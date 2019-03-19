// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package licenser

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs/elasticsearch"
)

// Enforce setups the corresponding callbacks in libbeat to verify the license on the
// remote elasticsearch cluster.
func Enforce(log *logp.Logger, checks ...CheckFunc) {
	cb := func(client *elasticsearch.Client) error {
		fetcher := NewElasticFetcher(client)
		license, err := fetcher.Fetch()

		if err != nil {
			return errors.Wrapf(err, "cannot retrieve the elasticsearch license or no license endpoint")
		}

		if license == OSSLicense {
			return errors.New(
				"The default distribution of Beats requires the default Elasticsearch distribution " +
					"(with Xpack included). Please either download the default Elasticsearch distribution " +
					"from elastic.co or the pure Apache 2.0 Beats distribution",
			)
		}

		if !Validate(log, *license, checks...) {
			return fmt.Errorf(
				"invalid license found, requires a basic or a valid trial license and received %s",
				license.Get(),
			)
		}

		return nil
	}

	elasticsearch.RegisterGlobalCallback(cb)
}
