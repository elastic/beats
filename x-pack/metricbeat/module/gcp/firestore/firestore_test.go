// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package firestore

import (
	"os"

	"github.com/elastic/beats/v8/metricbeat/mb"

	// Register input module and metricset
	_ "github.com/elastic/beats/v8/x-pack/metricbeat/module/gcp"
	_ "github.com/elastic/beats/v8/x-pack/metricbeat/module/gcp/metrics"
)

func init() {
	// To be moved to some kind of helper
	os.Setenv("BEAT_STRICT_PERMS", "false")
	mb.Registry.SetSecondarySource(mb.NewLightModulesSource("../../../module"))
}
