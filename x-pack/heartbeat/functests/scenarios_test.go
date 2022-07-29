package functests

import (
	"fmt"
	"github.com/elastic/beats/v7/heartbeat/hbtest"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"net/http/httptest"
	"net/url"
	"testing"
)

type Scenario struct {
	Name   string
	runner func() (config mapstr.M, close func(), err error)
}

func (s Scenario) Run(t *testing.T) (config mapstr.M, mtr *MonitorTestRun, err error) {
	config, close, err := s.runner()
	defer close()
	if err != nil {
		return nil, nil, err
	}

	mtr, err = runMonitorOnce(t, config)
	mtr.Wait()
	return config, mtr, err
}

var SimpleHTTPScenario = Scenario{
	Name: "http-simple",
	runner: func() (config mapstr.M, close func(), err error) {
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
}

var SimpleTCPScenario = Scenario{
	Name: "tcp-simple",
	runner: func() (config mapstr.M, close func(), err error) {
		server := httptest.NewServer(hbtest.HelloWorldHandler(200))
		parsedUrl, err := url.Parse(server.URL)
		if err != nil {
			panic(fmt.Sprintf("URL %s should always be parsable: %s", server.URL, err))
		}
		config = mapstr.M{
			"id":       "tcp-test-id",
			"name":     "tcp-test-name",
			"type":     "icmp",
			"schedule": "@every 1m",
			"hosts":    []string{fmt.Sprintf("%s:%s", parsedUrl.Host, parsedUrl.Port())},
		}
		return config, server.Close, nil
	},
}

var SimpleICMPScenario = Scenario{
	Name: "simple-icmp",
	runner: func() (config mapstr.M, close func(), err error) {
		return mapstr.M{
			"id":       "icmp-test-id",
			"name":     "icmp-test-name",
			"type":     "icmp",
			"schedule": "@every 1m",
			"hosts":    []string{"127.0.0.1"},
		}, func() {}, nil
	},
}

var SimpleBrowserScenario = Scenario{
	Name: "simple-browser",
	runner: func() (config mapstr.M, close func(), err error) {
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
}
