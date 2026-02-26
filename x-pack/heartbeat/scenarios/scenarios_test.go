// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package scenarios

import (
	"fmt"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/monitorstate"
	"github.com/elastic/beats/v7/x-pack/heartbeat/scenarios/framework"
)

var scenarioDB = framework.NewScenarioDB()
var testWs *httptest.Server
var failingTestWs *httptest.Server

// Note, no browser scenarios here, those all go in browserscenarios.go
// since they have different build tags
func init() {
	scenarioDB.Add(
		framework.Scenario{
			Name: "http-simple",
			Type: "http",
			Tags: []string{"lightweight", "http", "up"},
			Runner: func(t *testing.T) (config mapstr.M, meta framework.ScenarioRunMeta, close func(), err error) {
				server := startTestWebserver(t)
				meta.URL, _ = url.Parse(server.URL)
				meta.Status = monitorstate.StatusUp
				config = mapstr.M{
					"id":       "http-test-id",
					"name":     "http-test-name",
					"type":     "http",
					"schedule": "@every 1m",
					"urls":     []string{server.URL},
				}
				return config, meta, nil, nil
			},
		},
		framework.Scenario{
			Name: "http-down",
			Type: "http",
			Tags: []string{"lightweight", "http", "down"},
			Runner: func(t *testing.T) (config mapstr.M, meta framework.ScenarioRunMeta, close func(), err error) {
				server := startFailingTestWebserver(t)
				u := server.URL
				meta.URL, _ = url.Parse(u)
				meta.Status = monitorstate.StatusDown
				config = mapstr.M{
					"id":       "http-test-id",
					"name":     "http-test-name",
					"type":     "http",
					"schedule": "@every 1m",
					"urls":     []string{u},
				}
				return config, meta, nil, nil
			},
		},
		framework.Scenario{
			Name: "tcp-simple",
			Type: "tcp",
			Tags: []string{"lightweight", "tcp", "up"},
			Runner: func(t *testing.T) (config mapstr.M, meta framework.ScenarioRunMeta, close func(), err error) {
				server := startTestWebserver(t)
				parsedUrl, err := url.Parse(server.URL)
				if err != nil {
					panic(fmt.Sprintf("URL %s should always be parsable: %s", server.URL, err))
				}
				parsedUrl.Scheme = "tcp"
				meta.URL = parsedUrl
				meta.Status = monitorstate.StatusUp
				config = mapstr.M{
					"id":       "tcp-test-id",
					"name":     "tcp-test-name",
					"type":     "tcp",
					"schedule": "@every 1m",
					"hosts":    []string{parsedUrl.Host}, // Host includes host:port
				}
				return config, meta, nil, nil
			},
		},
		framework.Scenario{
			Name: "tcp-down",
			Type: "tcp",
			Tags: []string{"lightweight", "tcp", "down"},
			Runner: func(t *testing.T) (config mapstr.M, meta framework.ScenarioRunMeta, close func(), err error) {
				// This ip should never route anywhere
				// see https://stackoverflow.com/questions/528538/non-routable-ip-address
				parsedUrl, _ := url.Parse("tcp://192.0.2.0:8282")
				parsedUrl.Scheme = "tcp"
				meta.URL = parsedUrl
				meta.Status = monitorstate.StatusDown
				config = mapstr.M{
					"id":       "tcp-test-id",
					"name":     "tcp-test-name",
					"type":     "tcp",
					"schedule": "@every 1m",
					"hosts":    []string{parsedUrl.Host}, // Host includes host:port
				}
				return config, meta, nil, nil
			},
		},
		framework.Scenario{
			Name: "simple-icmp",
			Type: "icmp",
			Tags: []string{"icmp", "up"},
			Runner: func(t *testing.T) (config mapstr.M, meta framework.ScenarioRunMeta, close func(), err error) {
				meta.URL, _ = url.Parse("icp://127.0.0.1")
				meta.Status = monitorstate.StatusUp
				return mapstr.M{
					"id":       "icmp-test-id",
					"name":     "icmp-test-name",
					"type":     "icmp",
					"schedule": "@every 1m",
					"hosts":    []string{"127.0.0.1"},
				}, meta, nil, nil
			},
		},
	)
}
