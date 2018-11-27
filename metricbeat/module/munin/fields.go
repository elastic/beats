// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package munin

import (
	"github.com/elastic/beats/libbeat/asset"
)

func init() {
	if err := asset.SetFields("metricbeat", "munin", Asset); err != nil {
		panic(err)
	}
}

// Asset returns asset data
func Asset() string {
	return "eJxsjk2qhDAQhPc5ReHeC2TxbvAOEU2NNKOdkLQw3n4wBmaEqV3/FN834snDY9tV1AEmttJj+D/nwQGRdS6STZJ6/DkAaDdoisRGKzJX8JVTMRYHFK4MlR4TLTjgIVxj9a05QsPGD+2MHZkeS0l77psfyCuthjmpBdF6g1dGTAdC/2lyYaFar39bXCZ3z3cAAAD//xiKUG4="
}
