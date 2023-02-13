// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package main

import (
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/elastic/beats/v7/metricbeat/include"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/scripts/msetlists"
	"github.com/elastic/elastic-agent-libs/paths"
)

func main() {
	// Disable permission checks so it reads light modules in any case
	os.Setenv("BEAT_STRICT_PERMS", "false")

	path := paths.Resolve(paths.Home, "../metricbeat/module")
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
