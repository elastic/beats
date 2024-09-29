// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package cel_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
	filebeat "github.com/elastic/beats/v7/x-pack/filebeat/cmd"
	"github.com/elastic/elastic-agent-client/v7/pkg/client/mock"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
)

// TestCheckinV2 is an integration test that checks that CEL input reports in
// the expected statuses. Specifically it configures a filebeat instance to
// run a CEL input with two streams as well as it monitors the reported state
// by spawning an elastic-agent V2 mock server.
// This test also spawns two http servers for making the CEL input streams
// to report different states that are checked to match the expected states.
func TestCheckinV2(t *testing.T) {
	// make sure there is an ES instance running
	integration.EnsureESIsRunning(t)
	esConnectionDetails := integration.GetESURL(t, "http")
	outputHosts := []interface{}{fmt.Sprintf("%s://%s:%s", esConnectionDetails.Scheme, esConnectionDetails.Hostname(), esConnectionDetails.Port())}
	outputUsername := esConnectionDetails.User.Username()
	outputPassword, _ := esConnectionDetails.User.Password()
	outputProtocol := esConnectionDetails.Scheme

	invalidResponse := []byte("invalid json")
	validResponse := []byte("{\"ip\":\"0.0.0.0\"}")

	// http server for the first CEL input stream
	serverOneResponse := validResponse
	svrOne := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(serverOneResponse)
	}))
	defer svrOne.Close()

	// http server for the second CEL input stream
	serverTwoResponse := validResponse
	svrTwo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(serverTwoResponse)
	}))
	defer svrTwo.Close()

	// allStreams is an elastic-agent configuration with an ES output and one CEL
	// input with two streams.
	allStreams := []*proto.UnitExpected{
		{
			Id:             "output-unit",
			Type:           proto.UnitType_OUTPUT,
			ConfigStateIdx: 0,
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
					"revision":   0,
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

	// oneStream is an elastic-agent configuration with an ES output and one CEL
	// input with one stream. Effectively this is the same as allStreams with
	// stream cel-cel.cel-1e8b33de-d54a-45cd-90da-ffffffc482e2 removed.
	oneStream := []*proto.UnitExpected{
		{
			Id:             "output-unit",
			Type:           proto.UnitType_OUTPUT,
			ConfigStateIdx: 0,
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
					"revision":   0,
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

	// noStream is an elastic-agent configuration with just an ES output.
	noStream := []*proto.UnitExpected{
		{
			Id:             "output-unit",
			Type:           proto.UnitType_OUTPUT,
			ConfigStateIdx: 0,
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

	// elastic-agent management V2 mock server
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
	if err := server.Start(); err != nil {
		t.Fatalf("failed to start StubServerV2 server: %v", err)
	}
	defer server.Stop()

	// It's necessary to change os.Args so filebeat.Filebeat() can read the
	// appropriate args at beat.Execute().
	initialOSArgs := os.Args
	os.Args = []string{
		"filebeat",
		"-E", fmt.Sprintf(`management.insecure_grpc_url_for_testing="localhost:%d"`, server.Port),
		"-E", "management.enabled=true",
		"-E", "management.restart_on_output_change=true",
	}
	defer func() {
		os.Args = initialOSArgs
	}()

	beat := filebeat.Filebeat()
	beatRunErr := make(chan error)
	go func() {
		defer close(beatRunErr)
		beatRunErr <- beat.Execute()
	}()

	// slice of funcs that check if the observed states match the expected ones.
	// They return true if they match and false if they don't as well as a slice
	// of units expected for the server to respond with.
	checks := []func(t *testing.T, observed *proto.CheckinObserved) (bool, []*proto.UnitExpected){
		func(t *testing.T, observed *proto.CheckinObserved) (bool, []*proto.UnitExpected) {
			// Wait for all healthy.
			unitState, payload := extractStateAndPayload(observed, "input-unit-1")
			if unitState != proto.State_HEALTHY {
				return false, allStreams
			}

			if !reflect.DeepEqual(map[string]interface{}{
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

			return true, allStreams
		},
		func(t *testing.T, observed *proto.CheckinObserved) (bool, []*proto.UnitExpected) {
			// Wait for one degraded.
			unitState, payload := extractStateAndPayload(observed, "input-unit-1")
			if unitState != proto.State_DEGRADED {
				return false, allStreams
			}

			if !reflect.DeepEqual(map[string]interface{}{
				"streams": map[string]interface{}{
					"cel-cel.cel-1e8b33de-d54a-45cd-90da-23ed71c482e2": map[string]interface{}{
						"status": "DEGRADED",
						"error":  "failed evaluation: failed eval: ERROR: <input>:1:63: failed to unmarshal JSON message: invalid character 'i' looking for beginning of value\n | bytes(get(state.url).Body).as(body,{\"events\":[body.decode_json()]})\n | ..............................................................^",
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
			// Wait for all degraded.
			unitState, payload := extractStateAndPayload(observed, "input-unit-1")
			if unitState != proto.State_DEGRADED {
				return false, allStreams
			}

			if !reflect.DeepEqual(map[string]interface{}{
				"streams": map[string]interface{}{
					"cel-cel.cel-1e8b33de-d54a-45cd-90da-23ed71c482e2": map[string]interface{}{
						"status": "DEGRADED",
						"error":  "failed evaluation: failed eval: ERROR: <input>:1:63: failed to unmarshal JSON message: invalid character 'i' looking for beginning of value\n | bytes(get(state.url).Body).as(body,{\"events\":[body.decode_json()]})\n | ..............................................................^",
					},
					"cel-cel.cel-1e8b33de-d54a-45cd-90da-ffffffc482e2": map[string]interface{}{
						"status": "DEGRADED",
						"error":  "failed evaluation: failed eval: ERROR: <input>:1:63: failed to unmarshal JSON message: invalid character 'i' looking for beginning of value\n | bytes(get(state.url).Body).as(body,{\"events\":[body.decode_json()]})\n | ..............................................................^",
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
			// Wait for all healthy.
			unitState, payload := extractStateAndPayload(observed, "input-unit-1")
			if unitState != proto.State_HEALTHY {
				return false, allStreams
			}

			if !reflect.DeepEqual(map[string]interface{}{
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
			// Wait for all healthy.
			unitState, payload := extractStateAndPayload(observed, "input-unit-1")
			if unitState != proto.State_DEGRADED {
				return false, allStreams
			}

			if !reflect.DeepEqual(map[string]interface{}{
				"streams": map[string]interface{}{
					"cel-cel.cel-1e8b33de-d54a-45cd-90da-23ed71c482e2": map[string]interface{}{
						"status": "HEALTHY",
						"error":  "",
					},
					"cel-cel.cel-1e8b33de-d54a-45cd-90da-ffffffc482e2": map[string]interface{}{
						"status": "DEGRADED",
						"error":  "failed evaluation: failed eval: ERROR: <input>:1:63: failed to unmarshal JSON message: invalid character 'i' looking for beginning of value\n | bytes(get(state.url).Body).as(body,{\"events\":[body.decode_json()]})\n | ..............................................................^",
					},
				},
			}, payload) {
				return false, allStreams
			}

			setInputUnitsConfigStateIdx(oneStream, 1)
			return true, oneStream
		},
		func(t *testing.T, observed *proto.CheckinObserved) (bool, []*proto.UnitExpected) {
			unitState, payload := extractStateAndPayload(observed, "input-unit-1")
			if unitState != proto.State_HEALTHY {
				return false, oneStream
			}

			if !reflect.DeepEqual(map[string]interface{}{
				"streams": map[string]interface{}{
					"cel-cel.cel-1e8b33de-d54a-45cd-90da-23ed71c482e2": map[string]interface{}{
						"status": "HEALTHY",
						"error":  "",
					},
				},
			}, payload) {
				return false, oneStream
			}
			setInputUnitsConfigStateIdx(noStream, 2)
			return true, noStream
		},
		func(t *testing.T, observed *proto.CheckinObserved) (bool, []*proto.UnitExpected) {
			_, payload := extractStateAndPayload(observed, "input-unit-1")
			if payload != nil {
				return false, noStream
			}

			serverOneResponse = validResponse
			serverTwoResponse = validResponse

			setInputUnitsConfigStateIdx(allStreams, 3)
			return true, allStreams
		},
		func(t *testing.T, observed *proto.CheckinObserved) (bool, []*proto.UnitExpected) {
			// Wait for all healthy.
			unitState, payload := extractStateAndPayload(observed, "input-unit-1")
			if unitState != proto.State_HEALTHY {
				return false, allStreams
			}

			if !reflect.DeepEqual(map[string]interface{}{
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

			setInputUnitsConfigStateIdx(noStream, 4)
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

	const wait = 3 * time.Minute
	timer := time.NewTimer(wait)
	defer timer.Stop()
	for len(checks) > 0 {
		select {
		case observed := <-observedStates:
			matched, expected := checks[0](t, observed)
			expectedUnits <- expected
			if !matched {
				continue
			}
			timer.Reset(wait)
			checks = checks[1:]
		case err := <-beatRunErr:
			if err != nil {
				t.Fatalf("beat run err: %v", err)
			}
		case <-timer.C:
			t.Fatal("timeout waiting for checkin")
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

func setInputUnitsConfigStateIdx(units []*proto.UnitExpected, idx uint64) {
	for _, unit := range units {
		if unit.Type != proto.UnitType_INPUT {
			continue
		}

		if unit.Config == nil {
			return
		}
		unit.ConfigStateIdx = idx
		unit.Config.Source.Fields["revision"] = structpb.NewNumberValue(float64(idx))
	}
}
