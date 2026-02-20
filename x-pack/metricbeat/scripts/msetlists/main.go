// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/elastic/beats/v7/metricbeat/scripts/msetlists"

	"github.com/elastic/beats/v7/metricbeat/mb"
	_ "github.com/elastic/beats/v7/x-pack/metricbeat/include"
	"github.com/elastic/elastic-agent-libs/logp"
)

func main() {
	modulePath := flag.String(
		"module",
		"../x-pack/metricbeat/module",
		"Path to Metricbeat module directory",
	)

	flag.Parse()

	// Disable permission checks so it reads light modules in any case
	os.Setenv("BEAT_STRICT_PERMS", "false")

	lm := mb.NewLightModulesSource(logp.NewNopLogger(), *modulePath)
	mb.Registry.SetSecondarySource(lm)

	msList := msetlists.DefaultMetricsets()

	raw, err := json.MarshalIndent(msList, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error Marshalling json: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s\n", string(raw))
}
