// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package include

import (
	"github.com/elastic/beats/libbeat/feature"
	"github.com/elastic/beats/x-pack/functionbeat/manager/aws"
	"github.com/elastic/beats/x-pack/functionbeat/manager/gcp"
	"github.com/elastic/beats/x-pack/functionbeat/provider/local/local"
)

// Bundle feature enabled.
var Bundle = feature.MustBundle(
	aws.Bundle,
	gcp.Bundle,
	local.Bundle,
)

func init() {
	feature.MustRegisterBundle(Bundle)
}
