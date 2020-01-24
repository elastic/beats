// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package licenser

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs/elasticsearch"
)

const licenseDebugK = "license"

// Enforce setups the corresponding callbacks in libbeat to verify the license on the
// remote elasticsearch cluster.
func Enforce(name string, checks ...CheckFunc) {
	name = strings.Title(name)

	cb := func(client *elasticsearch.Client) error {
		log := logp.NewLogger(licenseDebugK)

		fetcher := NewElasticFetcher(client)
		license, err := fetcher.Fetch()

		if err != nil {
			return errors.Wrapf(err, "cannot retrieve the elasticsearch license from the /_xpack endpoint, "+
				"%s requires the default distribution of Elasticsearch. Please make the endpoint accessible "+
				"to %s so it can verify the license.", name, name)
		}

		if license == OSSLicense {
			return errors.Errorf("%s requires the default distribution of Elasticsearch. Please "+
				"update to the default distribution of Elasticsearch for full access to all "+
				"free features, or switch to the OSS distribution of %s.", name, name)
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
