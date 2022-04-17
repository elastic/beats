// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/menderesk/beats/v7/metricbeat/scripts/msetlists"

	"github.com/menderesk/beats/v7/libbeat/paths"
	"github.com/menderesk/beats/v7/metricbeat/mb"
	_ "github.com/menderesk/beats/v7/x-pack/metricbeat/include"
)

func main() {
	// Disable permission checks so it reads light modules in any case
	os.Setenv("BEAT_STRICT_PERMS", "false")

	path := paths.Resolve(paths.Home, "../x-pack/metricbeat/module")
	lm := mb.NewLightModulesSource(path)
	mb.Registry.SetSecondarySource(lm)

	msList := msetlists.DefaultMetricsets()

	raw, err := json.MarshalIndent(msList, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error Marshalling json: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s\n", string(raw))
}
