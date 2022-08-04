// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package scenarios

import (
	"fmt"
	"net/http/httptest"
	"net/url"
	"os"
	"sync"

	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/elastic/beats/v7/heartbeat/hbtest"
)

var Scenarios = &ScenarioDB{
	initOnce: &sync.Once{},
	ByTag:    map[string][]Scenario{},
	All: []Scenario{
		{
			Name: "http-simple",
			Type: "http",
			Tags: []string{"lightweight", "http"},
			Runner: func() (config mapstr.M, close func(), err error) {
				server := httptest.NewServer(hbtest.HelloWorldHandler(200))
				config = mapstr.M{
					"id":       "http-test-id",
					"name":     "http-test-name",
					"type":     "http",
					"schedule": "@every 1m",
					"urls":     []string{server.URL},
				}
				return config, server.Close, nil
			},
		},
		{
			Name: "tcp-simple",
			Type: "tcp",
			Tags: []string{"lightweight", "tcp"},
			Runner: func() (config mapstr.M, close func(), err error) {
				server := httptest.NewServer(hbtest.HelloWorldHandler(200))
				parsedUrl, err := url.Parse(server.URL)
				if err != nil {
					panic(fmt.Sprintf("URL %s should always be parsable: %s", server.URL, err))
				}
				config = mapstr.M{
					"id":       "tcp-test-id",
					"name":     "tcp-test-name",
					"type":     "tcp",
					"schedule": "@every 1m",
					"hosts":    []string{fmt.Sprintf("%s:%s", parsedUrl.Host, parsedUrl.Port())},
				}
				return config, server.Close, nil
			},
		},
		{
			Name: "simple-icmp",
			Type: "icmp",
			Tags: []string{"icmp"},
			Runner: func() (config mapstr.M, close func(), err error) {
				return mapstr.M{
					"id":       "icmp-test-id",
					"name":     "icmp-test-name",
					"type":     "icmp",
					"schedule": "@every 1m",
					"hosts":    []string{"127.0.0.1"},
				}, func() {}, nil
			},
		},
		{
			Name: "simple-browser",
			Type: "browser",
			Tags: []string{"browser", "browser-inline"},
			Runner: func() (config mapstr.M, close func(), err error) {
				err = os.Setenv("ELASTIC_SYNTHETICS_CAPABLE", "true")
				if err != nil {
					return nil, nil, err
				}
				server := httptest.NewServer(hbtest.HelloWorldHandler(200))
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
				return config, server.Close, nil
			},
		},
	},
}
