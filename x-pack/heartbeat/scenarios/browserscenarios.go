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
	_ "github.com/elastic/beats/v7/x-pack/heartbeat/monitors/browser"
	"github.com/elastic/beats/v7/x-pack/heartbeat/scenarios/framework"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func init() {
	scenarioDB.Add(
		framework.Scenario{
			Name: "simple-browser",
			Type: "browser",
			Tags: []string{"browser", "browser-inline"},
			Runner: func(t *testing.T) (config mapstr.M, meta framework.ScenarioRunMeta, close func(), err error) {
				err = os.Setenv("ELASTIC_SYNTHETICS_CAPABLE", "true")
				if err != nil {
					return nil, meta, nil, err
				}
				server := startTestWebserver(t)

				// Add / to normalize with test output
				meta.URL, _ = url.Parse(server.URL + "/")
				meta.Status = monitorstate.StatusUp
				config = mapstr.M{
					"id":       "browser-test-id",
					"name":     "browser-test-name",
					"type":     "browser",
					"schedule": "@every 1m",
					"source": mapstr.M{
						"inline": mapstr.M{
							"script": fmt.Sprintf("step('load server', async () => {await page.goto('%s')})", server.URL),
						},
					},
				}
				return config, meta, nil, nil
			},
		},
		framework.Scenario{
			Name: "failing-browser",
			Type: "browser",
			Tags: []string{"browser", "browser-inline", "down", "browser-down"},
			Runner: func(t *testing.T) (config mapstr.M, meta framework.ScenarioRunMeta, close func(), err error) {
				err = os.Setenv("ELASTIC_SYNTHETICS_CAPABLE", "true")
				if err != nil {
					return nil, meta, nil, err
				}
				server := startTestWebserver(t)

				// Add / to normalize with test output
				meta.URL, _ = url.Parse(server.URL + "/")
				meta.Status = monitorstate.StatusDown
				config = mapstr.M{
					"id":       "browser-test-id",
					"name":     "browser-test-name",
					"type":     "browser",
					"schedule": "@every 1m",
					"source": mapstr.M{
						"inline": mapstr.M{
							"script": fmt.Sprintf("step('load server', async () => {await page.goto('%s'); throw(\"anerr\")})", meta.URL),
						},
					},
				}
				return config, meta, nil, nil
			},
		},
	)
}
