package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/elastic/beats/libbeat/paths"
	_ "github.com/elastic/beats/metricbeat/include"
	"github.com/elastic/beats/metricbeat/mb"
	_ "github.com/elastic/beats/x-pack/metricbeat/include"
	xpackmb "github.com/elastic/beats/x-pack/metricbeat/mb"
)

func main() {
	path := paths.Resolve(paths.Home, "../x-pack/metricbeat/module")
	lm := xpackmb.NewLightModulesSource(path)
	mb.Registry.SetSecondarySource(lm)

	// List all registered modules and metricsets.
	var defaultMap = make(map[string][]string)
	for _, mod := range mb.Registry.Modules() {
		metricSets, err := mb.Registry.DefaultMetricSets(mod)
		if err != nil && !strings.Contains(err.Error(), "no default metricset for") {
			continue
		}
		defaultMap[mod] = metricSets
	}

	raw, err := json.MarshalIndent(defaultMap, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error Marshalling json: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s\n", string(raw))
}
