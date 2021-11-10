package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/elastic/beats/v7/heartbeat/config"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"testing"
	"time"
)

type MockClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

var (
	// GetDoFunc fetches the mock client's `Do` func
	GetDoFunc func(req *http.Request) (*http.Response, error)
)

func (m *MockClient) Do(req *http.Request) (*http.Response, error) {
	return GetDoFunc(req)
}

func init() {
	Client = &MockClient{}

	GetDoFunc = func(req *http.Request) (*http.Response, error) {
		return nil, errors.New(
			"error from web server",
		)
	}
}

type MapStr = common.MapStr

func MockMonitorConfig(t *testing.T, rawConf MapStr) (config.Config, *common.Config) {
	testHbConf := MapStr{
		"monitors": []MapStr{{
			"type":              "test",
			"urls":              []string{"https://google.com"},
			"schedule":          "@every 10m",
			"service_locations": []string{"us-east"},
		},
		},
		"service": MapStr{
			"username":     "admin",
			"password":     "changeme",
			"manifest_url": "http://localhost:8220",
		},
	}

	if rawConf != nil {
		testHbConf = rawConf
	}
	rawConfig, _ := common.NewConfigFrom(testHbConf)
	parsedConfig := config.DefaultConfig
	err1 := rawConfig.Unpack(&parsedConfig)
	if err1 != nil {
		t.Error(err1)
	}
	return parsedConfig, rawConfig
}

func MockSyntheticService(t *testing.T, rawConfig MapStr) *SyntheticsService {
	cfg, _ := MockMonitorConfig(t, rawConfig)
	return &SyntheticsService{
		config: cfg,
	}
}

func mockResponse() (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(bytes.NewBufferString("success")),
	}, nil
}

func TestPushConfiguration(t *testing.T) {
	sv := MockSyntheticService(t, nil)
	payload := SyntheticServicePayload{}
	GetDoFunc = func(req *http.Request) (*http.Response, error) {
		bd, _ := ioutil.ReadAll(req.Body)
		err := json.Unmarshal(bd, &payload)
		if err != nil {
			return nil, err
		}
		return mockResponse()
	}
	username := "elastic"
	password := "changeme"
	sv.servicePushWait.Add(1)
	sv.pushConfigsToSyntheticsService("us-east", config.ServiceLocation{
		Url: "http://localhost:8220/cronjob",
	}, Output{
		Hosts:    []string{"http:localhost:9200"},
		Username: username,
		Password: password,
	}, time.Millisecond)
	if len(payload.Monitors) != 1 {
		t.Error("expected payload monitors length to be 1")
	}
	assert.Equal(t, username, payload.Output.Username)
	assert.Equal(t, password, payload.Output.Password)
}

func TestPushConfigurationRetries(t *testing.T) {
	sv := MockSyntheticService(t, nil)
	numberOfTimeCalled := 0
	GetDoFunc = func(req *http.Request) (*http.Response, error) {
		numberOfTimeCalled++
		return nil, errors.New("error")
	}
	username := "elastic"
	password := "changeme"
	sv.servicePushWait.Add(1)
	sv.pushConfigsToSyntheticsService("us-east", config.ServiceLocation{
		Url: "http://localhost:8220/cronjob",
	}, Output{
		Hosts:    []string{"http:localhost:9200"},
		Username: username,
		Password: password,
	}, time.Millisecond)
	assert.Equal(t, numberOfTimeCalled, 3)
}

func TestRunViaSyntheticsService(t *testing.T) {
	testHbConf := MapStr{
		"monitors": []MapStr{{
			"type":              "test",
			"urls":              []string{"https://google.com"},
			"schedule":          "@every 10m",
			"service_locations": []string{"us_central"},
		},
		},
		"service": MapStr{
			"username":     "admin",
			"password":     "changeme",
			"manifest_url": "http://localhost:8220",
		},
	}
	sv := MockSyntheticService(t, testHbConf)
	bInfo := beat.Info{
		Beat:        "heartbeat",
		IndexPrefix: "heartbeat",
		Version:     "8.0.0",
	}
	username := "elastic"
	password := "changeme"
	hosts := []string{"http://localhost:9200"}
	cfgMap := common.MapStr{
		"hosts":    hosts,
		"username": username,
		"password": password,
	}
	cfg, _ := common.NewConfigFrom(cfgMap)
	output := common.NewConfigNameSpace("output", cfg)
	bConfig := beat.BeatConfig{Output: *output}
	b := beat.Beat{
		Info:   bInfo,
		Config: &bConfig,
	}
	payload := SyntheticServicePayload{}
	GetDoFunc = func(req *http.Request) (*http.Response, error) {
		if req.Body != nil {
			bd, _ := ioutil.ReadAll(req.Body)
			err := json.Unmarshal(bd, &payload)
			if err != nil {
				return nil, err
			}
			serviceUser, servicePass, _ := req.BasicAuth()

			assert.Equal(t, serviceUser, "admin")
			assert.Equal(t, servicePass, "changeme")

			return mockResponse()
		}
		jsonStr := `{
					  "locations": {
						"us_central": {
						  "url": "us-central.synthetics.elastic.dev",
						  "geo": {
							"name": "US Central",
							"location": {"lat": 41.25, "lon": -95.86}
						  },
						  "status": "beta"
						}
					  }
					}`
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte(jsonStr))),
		}, nil
	}
	err2 := sv.Run(&b)
	if err2 != nil {
		t.Error(err2)
	}
	defer sv.servicePushWait.Wait()

	// wait for go routine
	time.Sleep(time.Second * 5)

	if len(payload.Monitors) != 1 {
		t.Error("expected payload monitors length to be 1")
	}

	assert.Equal(t, username, payload.Output.Username)
	assert.Equal(t, password, payload.Output.Password)
	assert.Equal(t, hosts, payload.Output.Hosts)

	sv.servicePushWait.Done()

	sv.Stop()
}

func TestValidateMonitorsSchedule(t *testing.T) {
	inValidMonCfg := MapStr{
		"monitors": []MapStr{{
			"type":              "test",
			"urls":              []string{"https://google.com"},
			"schedule":          "@every 10s",
			"service_locations": []string{"us-east"},
		},
		},
	}
	sv := MockSyntheticService(t, inValidMonCfg)
	err := sv.validateMonitorsSchedule()
	if err == nil {
		t.Error("it should return error of an invalid monitor")
	}
	validMonCfg := MapStr{
		"monitors": []MapStr{{
			"type":              "test",
			"urls":              []string{"https://google.com"},
			"schedule":          "@every 10m",
			"service_locations": []string{"us-east"},
		},
		},
	}
	sv = MockSyntheticService(t, validMonCfg)
	err = sv.validateMonitorsSchedule()
	if err != nil {
		t.Error("it should not return an error of a valid monitor")
	}
}

func TestGetServiceManifest(t *testing.T) {
	sv := MockSyntheticService(t, nil)
	GetDoFunc = func(req *http.Request) (*http.Response, error) {
		jsonStr := `{
				  "locations": {
					"us_central": {
					  "url": "us-central.synthetics.elastic.dev",
					  "geo": {
						"name": "US Central",
						"location": {"lat": 41.25, "lon": -95.86}
					  },
					  "status": "beta"
					}
				  }
				}`

		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte(jsonStr))),
		}, nil
	}
	manifest, _ := sv.getSyntheticServiceManifest()
	assert.Equal(t, len(manifest.Locations), 1)
	for locName, loc := range manifest.Locations {
		assert.Equal(t, locName, "us_central")
		assert.Equal(t, loc.Url, "us-central.synthetics.elastic.dev")
	}
}
