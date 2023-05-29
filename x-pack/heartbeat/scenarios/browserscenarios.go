// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux || darwin

package scenarios

import (
	"fmt"
	"os"
	"testing"

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
			Runner: func(t *testing.T) (config mapstr.M, close func(), err error) {
				err = os.Setenv("ELASTIC_SYNTHETICS_CAPABLE", "true")
				if err != nil {
					return nil, nil, err
				}
				server := startTestWebserver(t)
				config = mapstr.M{
					"id":       "browser-test-id",
					"name":     "browser-test-name",
					"type":     "browser",
					"schedule": "@every 1m",
					"hosts":    []string{"127.0.0.1"},
					"source": mapstr.M{
						"inline": mapstr.M{
							"script": fmt.Sprintf("step('load server', async () => {await page.goto('%s')})", server.URL),
						},
					},
				}
				return config, nil, nil
			},
		},
	)
}
