// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
	filebeat "github.com/elastic/beats/v7/x-pack/filebeat/cmd"
	"github.com/elastic/elastic-agent-client/v7/pkg/client/mock"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
)

func setUnitsRevisionNumber(units []*proto.UnitExpected, revision uint64) {
	for _, unit := range units {
		if unit.Config == nil {
			return
		}

		unit.Config.Revision = revision
	}
}

func TestCELCheckinV2(t *testing.T) {
	integration.EnsureESIsRunning(t)

	invalidResponse := []byte("invalid json")
	validResponse := []byte("{\"ip\":\"0.0.0.0\"}")

	serverOneResponse := validResponse
	svrOne := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(serverOneResponse)
	}))
	defer svrOne.Close()

	serverTwoResponse := validResponse
	svrTwo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(serverTwoResponse)
	}))
	defer svrTwo.Close()

	esConnectionDetails := integration.GetESURL(t, "http")
	outputHosts := []interface{}{fmt.Sprintf("%s://%s:%s", esConnectionDetails.Scheme, esConnectionDetails.Hostname(), esConnectionDetails.Port())}
	outputUsername := esConnectionDetails.User.Username()
	outputPassword, _ := esConnectionDetails.User.Password()
	outputProtocol := esConnectionDetails.Scheme

	allStreams := []*proto.UnitExpected{
		{
			Id:             "output-unit",
			Type:           proto.UnitType_OUTPUT,
			ConfigStateIdx: 1,
			State:          proto.State_HEALTHY,
			LogLevel:       proto.UnitLogLevel_INFO,
			Config: &proto.UnitExpectedConfig{
				Id:   "default",
				Type: "elasticsearch",
				Name: "elasticsearch",
				Source: integration.RequireNewStruct(t, map[string]interface{}{
					"type":                  "elasticsearch",
					"hosts":                 outputHosts,
					"username":              outputUsername,
					"password":              outputPassword,
					"protocol":              outputProtocol,
					"enabled":               true,
					"ssl.verification_mode": "none",
				}),
			},
		},
		{
			Id:             "input-unit-1",
			Type:           proto.UnitType_INPUT,
			ConfigStateIdx: 0,
			State:          proto.State_HEALTHY,
			LogLevel:       proto.UnitLogLevel_DEBUG,
			Config: &proto.UnitExpectedConfig{
				Id:   "cel-cel-1e8b33de-d54a-45cd-90da-23ed71c482e5",
				Type: "cel",
				Name: "cel-1",
				Source: integration.RequireNewStruct(t, map[string]interface{}{
					"use_output": "default",
					"revision":   1,
				}),
				DataStream: &proto.DataStream{
					Namespace: "default",
				},
				Meta: &proto.Meta{
					Package: &proto.Package{
						Name:    "cel",
						Version: "1.9.0",
					},
				},
				Streams: []*proto.Stream{
					{
						Id: "cel-cel.cel-1e8b33de-d54a-45cd-90da-23ed71c482e2",
						DataStream: &proto.DataStream{
							Dataset: "cel.cel",
						},
						Source: integration.RequireNewStruct(t, map[string]interface{}{
							"interval":                        "10s",
							"program":                         `bytes(get(state.url).Body).as(body,{"events":[body.decode_json()]})`,
							"redact.delete":                   false,
							"regexp":                          nil,
							"resource.url":                    svrOne.URL,
							"publisher_pipeline.disable_host": true,
						}),
					},
					{
						Id: "cel-cel.cel-1e8b33de-d54a-45cd-90da-ffffffc482e2",
						DataStream: &proto.DataStream{
							Dataset: "cel.cel",
						},
						Source: integration.RequireNewStruct(t, map[string]interface{}{
							"interval":                        "10s",
							"program":                         `bytes(get(state.url).Body).as(body,{"events":[body.decode_json()]})`,
							"redact.delete":                   false,
							"regexp":                          nil,
							"resource.url":                    svrTwo.URL,
							"publisher_pipeline.disable_host": true,
						}),
					},
				},
			},
		},
	}

	oneStream := []*proto.UnitExpected{
		{
			Id:             "output-unit",
			Type:           proto.UnitType_OUTPUT,
			ConfigStateIdx: 1,
			State:          proto.State_HEALTHY,
			LogLevel:       proto.UnitLogLevel_INFO,
			Config: &proto.UnitExpectedConfig{
				Id:   "default",
				Type: "elasticsearch",
				Name: "elasticsearch",
				Source: integration.RequireNewStruct(t, map[string]interface{}{
					"type":                  "elasticsearch",
					"hosts":                 outputHosts,
					"username":              outputUsername,
					"password":              outputPassword,
					"protocol":              outputProtocol,
					"enabled":               true,
					"ssl.verification_mode": "none",
				}),
			},
		},
		{
			Id:             "input-unit-1",
			Type:           proto.UnitType_INPUT,
			ConfigStateIdx: 0,
			State:          proto.State_HEALTHY,
			LogLevel:       proto.UnitLogLevel_DEBUG,
			Config: &proto.UnitExpectedConfig{
				Id:   "cel-cel-1e8b33de-d54a-45cd-90da-23ed71c482e5",
				Type: "cel",
				Name: "cel-1",
				Source: integration.RequireNewStruct(t, map[string]interface{}{
					"use_output": "default",
					"revision":   1,
				}),
				DataStream: &proto.DataStream{
					Namespace: "default",
				},
				Meta: &proto.Meta{
					Package: &proto.Package{
						Name:    "cel",
						Version: "1.9.0",
					},
				},
				Streams: []*proto.Stream{
					{
						Id: "cel-cel.cel-1e8b33de-d54a-45cd-90da-23ed71c482e2",
						DataStream: &proto.DataStream{
							Dataset: "cel.cel",
						},
						Source: integration.RequireNewStruct(t, map[string]interface{}{
							"interval":                        "10s",
							"program":                         `bytes(get(state.url).Body).as(body,{"events":[body.decode_json()]})`,
							"redact.delete":                   false,
							"regexp":                          nil,
							"resource.url":                    svrOne.URL,
							"publisher_pipeline.disable_host": true,
						}),
					},
				},
			},
		},
	}

	noStream := []*proto.UnitExpected{
		{
			Id:             "output-unit",
			Type:           proto.UnitType_OUTPUT,
			ConfigStateIdx: 1,
			State:          proto.State_HEALTHY,
			LogLevel:       proto.UnitLogLevel_INFO,
			Config: &proto.UnitExpectedConfig{
				Id:   "default",
				Type: "elasticsearch",
				Name: "elasticsearch",
				Source: integration.RequireNewStruct(t, map[string]interface{}{
					"type":                  "elasticsearch",
					"hosts":                 outputHosts,
					"username":              outputUsername,
					"password":              outputPassword,
					"protocol":              outputProtocol,
					"enabled":               true,
					"ssl.verification_mode": "none",
				}),
			},
		},
	}

	observedStates := make(chan *proto.CheckinObserved)
	expectedUnits := make(chan []*proto.UnitExpected)
	done := make(chan struct{})
	server := &mock.StubServerV2{
		CheckinV2Impl: func(observed *proto.CheckinObserved) *proto.CheckinExpected {
			select {
			case observedStates <- observed:
				return &proto.CheckinExpected{
					Units: <-expectedUnits,
				}
			case <-done:
				return nil
			}
		},
		ActionImpl: func(response *proto.ActionResponse) error {
			return nil
		},
	}
	defer server.Stop()

	require.NoError(t, server.Start())

	os.Args = []string{
		"filebeat",
		"-E", fmt.Sprintf(`management.insecure_grpc_url_for_testing="localhost:%d"`, server.Port),
		"-E", "management.enabled=true",
		"-E", "management.restart_on_output_change=true",
	}

	beat := filebeat.Filebeat()
	beatRunErr := make(chan error)
	go func() {
		defer close(beatRunErr)
		beatRunErr <- beat.Execute()
	}()

	checks := []func(t *testing.T, observed *proto.CheckinObserved) (bool, []*proto.UnitExpected){
		func(t *testing.T, observed *proto.CheckinObserved) (bool, []*proto.UnitExpected) {
			// wait for all healthy
			unitState, payload := extractStateAndPayload(observed, "input-unit-1")
			if unitState != proto.State_HEALTHY {
				return false, allStreams
			}

			if !assert.ObjectsAreEqualValues(map[string]interface{}{
				"streams": map[string]interface{}{
					"cel-cel.cel-1e8b33de-d54a-45cd-90da-23ed71c482e2": map[string]interface{}{
						"status": "HEALTHY",
						"error":  "",
					},
					"cel-cel.cel-1e8b33de-d54a-45cd-90da-ffffffc482e2": map[string]interface{}{
						"status": "HEALTHY",
						"error":  "",
					},
				},
			}, payload) {
				return false, allStreams
			}

			serverOneResponse = invalidResponse

			setUnitsRevisionNumber(allStreams, 0)
			return true, allStreams
		},
		func(t *testing.T, observed *proto.CheckinObserved) (bool, []*proto.UnitExpected) {
			// wait for one degraded
			unitState, payload := extractStateAndPayload(observed, "input-unit-1")
			if unitState != proto.State_DEGRADED {
				return false, allStreams
			}

			if !assert.ObjectsAreEqualValues(map[string]interface{}{
				"streams": map[string]interface{}{
					"cel-cel.cel-1e8b33de-d54a-45cd-90da-23ed71c482e2": map[string]interface{}{
						"status": "DEGRADED",
						"error":  "failed eval: ERROR: <input>:1:30: failed to unmarshal JSON message: invalid character 'i' looking for beginning of value\n | bytes(get(state.url).Body).as(body,{\"events\":[body.decode_json()]})\n | .............................^",
					},
					"cel-cel.cel-1e8b33de-d54a-45cd-90da-ffffffc482e2": map[string]interface{}{
						"status": "HEALTHY",
						"error":  "",
					},
				},
			}, payload) {
				return false, allStreams
			}

			serverTwoResponse = invalidResponse
			return true, allStreams
		},
		func(t *testing.T, observed *proto.CheckinObserved) (bool, []*proto.UnitExpected) {
			// wait for all degraded
			unitState, payload := extractStateAndPayload(observed, "input-unit-1")
			if unitState != proto.State_DEGRADED {
				return false, allStreams
			}

			if !assert.ObjectsAreEqualValues(map[string]interface{}{
				"streams": map[string]interface{}{
					"cel-cel.cel-1e8b33de-d54a-45cd-90da-23ed71c482e2": map[string]interface{}{
						"status": "DEGRADED",
						"error":  "failed eval: ERROR: <input>:1:30: failed to unmarshal JSON message: invalid character 'i' looking for beginning of value\n | bytes(get(state.url).Body).as(body,{\"events\":[body.decode_json()]})\n | .............................^",
					},
					"cel-cel.cel-1e8b33de-d54a-45cd-90da-ffffffc482e2": map[string]interface{}{
						"status": "DEGRADED",
						"error":  "failed eval: ERROR: <input>:1:30: failed to unmarshal JSON message: invalid character 'i' looking for beginning of value\n | bytes(get(state.url).Body).as(body,{\"events\":[body.decode_json()]})\n | .............................^",
					},
				},
			}, payload) {
				return false, allStreams
			}

			serverOneResponse = validResponse
			serverTwoResponse = validResponse
			return true, allStreams
		},
		func(t *testing.T, observed *proto.CheckinObserved) (bool, []*proto.UnitExpected) {
			// wait for all healthy
			unitState, payload := extractStateAndPayload(observed, "input-unit-1")
			if unitState != proto.State_HEALTHY {
				return false, allStreams
			}

			if !assert.ObjectsAreEqualValues(map[string]interface{}{
				"streams": map[string]interface{}{
					"cel-cel.cel-1e8b33de-d54a-45cd-90da-23ed71c482e2": map[string]interface{}{
						"status": "HEALTHY",
						"error":  "",
					},
					"cel-cel.cel-1e8b33de-d54a-45cd-90da-ffffffc482e2": map[string]interface{}{
						"status": "HEALTHY",
						"error":  "",
					},
				},
			}, payload) {
				return false, allStreams
			}

			serverTwoResponse = invalidResponse
			return true, allStreams
		},
		func(t *testing.T, observed *proto.CheckinObserved) (bool, []*proto.UnitExpected) {
			// wait for all healthy
			unitState, payload := extractStateAndPayload(observed, "input-unit-1")
			if unitState != proto.State_DEGRADED {
				return false, allStreams
			}

			if !assert.ObjectsAreEqualValues(map[string]interface{}{
				"streams": map[string]interface{}{
					"cel-cel.cel-1e8b33de-d54a-45cd-90da-23ed71c482e2": map[string]interface{}{
						"status": "HEALTHY",
						"error":  "",
					},
					"cel-cel.cel-1e8b33de-d54a-45cd-90da-ffffffc482e2": map[string]interface{}{
						"status": "DEGRADED",
						"error":  "failed eval: ERROR: <input>:1:30: failed to unmarshal JSON message: invalid character 'i' looking for beginning of value\n | bytes(get(state.url).Body).as(body,{\"events\":[body.decode_json()]})\n | .............................^",
					},
				},
			}, payload) {
				return false, allStreams
			}

			setUnitsRevisionNumber(oneStream, 1)
			return true, oneStream
		},
		func(t *testing.T, observed *proto.CheckinObserved) (bool, []*proto.UnitExpected) {
			unitState, payload := extractStateAndPayload(observed, "input-unit-1")
			if unitState != proto.State_HEALTHY {
				return false, oneStream
			}

			if !assert.ObjectsAreEqualValues(map[string]interface{}{
				"streams": map[string]interface{}{
					"cel-cel.cel-1e8b33de-d54a-45cd-90da-23ed71c482e2": map[string]interface{}{
						"status": "HEALTHY",
						"error":  "",
					},
				},
			}, payload) {
				return false, oneStream
			}
			setUnitsRevisionNumber(noStream, 2)
			return true, noStream
		},
		func(t *testing.T, observed *proto.CheckinObserved) (bool, []*proto.UnitExpected) {
			_, payload := extractStateAndPayload(observed, "input-unit-1")
			if payload != nil {
				return false, noStream
			}

			serverOneResponse = validResponse
			serverTwoResponse = validResponse

			setUnitsRevisionNumber(allStreams, 3)
			return true, allStreams
		},
		func(t *testing.T, observed *proto.CheckinObserved) (bool, []*proto.UnitExpected) {
			// wait for all healthy
			unitState, payload := extractStateAndPayload(observed, "input-unit-1")
			if unitState != proto.State_HEALTHY {
				return false, allStreams
			}

			if !assert.ObjectsAreEqualValues(map[string]interface{}{
				"streams": map[string]interface{}{
					"cel-cel.cel-1e8b33de-d54a-45cd-90da-23ed71c482e2": map[string]interface{}{
						"status": "HEALTHY",
						"error":  "",
					},
					"cel-cel.cel-1e8b33de-d54a-45cd-90da-ffffffc482e2": map[string]interface{}{
						"status": "HEALTHY",
						"error":  "",
					},
				},
			}, payload) {
				return false, allStreams
			}

			setUnitsRevisionNumber(noStream, 4)
			return true, noStream
		},
		func(t *testing.T, observed *proto.CheckinObserved) (bool, []*proto.UnitExpected) {
			_, payload := extractStateAndPayload(observed, "input-unit-1")
			if payload != nil {
				return false, noStream
			}

			return true, []*proto.UnitExpected{}
		},
		func(t *testing.T, observed *proto.CheckinObserved) (bool, []*proto.UnitExpected) {
			return len(observed.Units) == 0, []*proto.UnitExpected{}
		},
	}

	timer := time.NewTimer(3 * time.Minute)
	defer timer.Stop()
	for {
		select {
		case observed := <-observedStates:
			t.Logf("observed: %v", observed)
			next, expected := checks[0](t, observed)

			expectedUnits <- expected

			if !next {
				continue
			}

			timer.Reset(3 * time.Minute)

			if len(checks) > 0 {
				checks = checks[1:]
			}

			if len(checks) == 0 {
				return
			}
		case err := <-beatRunErr:
			require.NoError(t, err)
		case <-timer.C:
			require.FailNow(t, "timeout waiting for checkin")
		}
	}

}

func extractStateAndPayload(observed *proto.CheckinObserved, inputID string) (proto.State, map[string]interface{}) {

	for _, unit := range observed.GetUnits() {
		if unit.Id == inputID {
			return unit.GetState(), unit.Payload.AsMap()
		}
	}

	return -1, nil
}
