// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package rds

import (
	"os"

	"github.com/elastic/beats/v9/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/logp"

	// Register input module and metricset
	_ "github.com/elastic/beats/v9/x-pack/metricbeat/module/aws"
	_ "github.com/elastic/beats/v9/x-pack/metricbeat/module/aws/cloudwatch"
)

func init() {
	// To be moved to some kind of helper
	os.Setenv("BEAT_STRICT_PERMS", "false")
	mb.Registry.SetSecondarySource(mb.NewLightModulesSource(logp.NewNopLogger(), "../../../module"))
}
