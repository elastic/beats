// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package include

import (
	// Register Fleet
	_ "github.com/elastic/beats/v7/x-pack/libbeat/management"

<<<<<<< HEAD:x-pack/libbeat/cmd/inject.go
	// Register fleet
	_ "github.com/elastic/beats/v7/x-pack/libbeat/management/fleet"

=======
>>>>>>> 27e76c567 (Remove Beats central management (#25696)):x-pack/libbeat/include/include.go
	// register processors
	_ "github.com/elastic/beats/v7/x-pack/libbeat/processors/add_cloudfoundry_metadata"
	_ "github.com/elastic/beats/v7/x-pack/libbeat/processors/add_nomad_metadata"

	// register autodiscover providers
	_ "github.com/elastic/beats/v7/x-pack/libbeat/autodiscover/providers/aws/ec2"
	_ "github.com/elastic/beats/v7/x-pack/libbeat/autodiscover/providers/aws/elb"
	_ "github.com/elastic/beats/v7/x-pack/libbeat/autodiscover/providers/nomad"
)
