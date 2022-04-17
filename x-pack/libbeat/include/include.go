// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package include

import (
	// Register Fleet
	_ "github.com/menderesk/beats/v7/x-pack/libbeat/management"

	// register processors
	_ "github.com/menderesk/beats/v7/x-pack/libbeat/processors/add_cloudfoundry_metadata"
	_ "github.com/menderesk/beats/v7/x-pack/libbeat/processors/add_nomad_metadata"

	// register autodiscover providers
	_ "github.com/menderesk/beats/v7/x-pack/libbeat/autodiscover/providers/aws/ec2"
	_ "github.com/menderesk/beats/v7/x-pack/libbeat/autodiscover/providers/aws/elb"
	_ "github.com/menderesk/beats/v7/x-pack/libbeat/autodiscover/providers/nomad"
)
