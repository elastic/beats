// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux || darwin || synthetics

package scenarios

import (
	"fmt"
	"net/url"
	"os"
	"testing"

	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/monitorstate"
	_ "github.com/elastic/beats/v7/x-pack/heartbeat/monitors/api"
	"github.com/elastic/beats/v7/x-pack/heartbeat/scenarios/framework"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// A module-style inline source (leading import) is loaded as-is by the agent,
// so apiJourney registers a real API journey instead of the implicit browser
// journey a bare script would produce.
const apiInlineJourney = `import { apiJourney, step } from '@elastic/synthetics';
apiJourney('api journey', ({ request }) => {
  step('get server', async () => {
    await request.get('%s');
  });
});`

func init() {
	scenarioDB.Add(
		framework.Scenario{
			Name: "simple-api",
			Type: "api",
			Tags: []string{"api", "api-inline"},
			Runner: func(t *testing.T) (config mapstr.M, meta framework.ScenarioRunMeta, close func(), err error) {
				if err = os.Setenv("ELASTIC_SYNTHETICS_CAPABLE", "true"); err != nil {
					return nil, meta, nil, err
				}
				server := startTestWebserver(t)

				meta.URL, _ = url.Parse(server.URL + "/")
				meta.Status = monitorstate.StatusUp
				config = mapstr.M{
					"id":       "api-test-id",
					"name":     "api-test-name",
					"type":     "api",
					"schedule": "@every 1m",
					"source": mapstr.M{
						"inline": mapstr.M{
							"script": fmt.Sprintf(apiInlineJourney, server.URL),
						},
					},
				}
				return config, meta, nil, nil
			},
		},
	)
}
