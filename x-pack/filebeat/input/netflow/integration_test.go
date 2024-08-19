// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package netflow_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"golang.org/x/time/rate"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/tests/integration"
	filebeat "github.com/elastic/beats/v7/x-pack/filebeat/cmd"
	"github.com/elastic/elastic-agent-client/v7/pkg/client/mock"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
	"github.com/elastic/elastic-agent-libs/monitoring"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"github.com/stretchr/testify/require"
)

const (
	waitFor = 10 * time.Second
	tick    = 200 * time.Millisecond
)

func TestNetFlowIntegration(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// make sure there is an ES instance running
	integration.EnsureESIsRunning(t)
	esConnectionDetails := integration.GetESURL(t, "http")
	outputHost := fmt.Sprintf("%s://%s:%s", esConnectionDetails.Scheme, esConnectionDetails.Hostname(), esConnectionDetails.Port())
	outputHosts := []interface{}{outputHost}

	// we are going to need admin access to query ES about the logs-netflow.log-default data_stream
	outputUsername := os.Getenv("ES_SUPERUSER_USER")
	require.NotEmpty(t, outputUsername)
	outputPassword := os.Getenv("ES_SUPERUSER_PASS")
	require.NotEmpty(t, outputPassword)
	outputProtocol := esConnectionDetails.Scheme

	deleted, err := DeleteDataStream(ctx, outputUsername, outputPassword, outputHost, "logs-netflow.log-default")
	require.NoError(t, err)
	require.True(t, deleted)

	// construct expected Agent units
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
					// ref: https://www.elastic.co/guide/en/fleet/8.14/es-output-settings.html
					"preset":                     "custom",
					"bulk_max_size":              1600,
					"worker":                     4,
					"queue.mem.events":           12800,
					"queue.mem.flush.min_events": 1600,
					"queue.mem.flush.timeout":    5,
					"compression_level":          1,
					"connection_idle_timeout":    15,
				}),
			},
		},
		{
			Id:             "input-unit-1",
			Type:           proto.UnitType_INPUT,
			ConfigStateIdx: 0,
			State:          proto.State_HEALTHY,
			LogLevel:       proto.UnitLogLevel_INFO,
			Config: &proto.UnitExpectedConfig{
				Id:   "netflow-netflow-1e8b33de-d54a-45cd-90da-23ed71c482e5",
				Type: "netflow",
				Name: "netflow-1",
				Source: integration.RequireNewStruct(t, map[string]interface{}{
					"use_output": "default",
					"revision":   0,
				}),
				DataStream: &proto.DataStream{
					Namespace: "default",
				},
				Meta: &proto.Meta{
					Package: &proto.Package{
						Name:    "netflow",
						Version: "1.9.0",
					},
				},
				Streams: []*proto.Stream{
					{
						Id: "netflow-netflow.netflow-1e8b33de-d54a-45cd-90da-23ed71c482e2",
						DataStream: &proto.DataStream{
							Dataset: "netflow.log",
						},
						Source: integration.RequireNewStruct(t, map[string]interface{}{
							"id":                    "netflow_integration_test",
							"host":                  "localhost:6006",
							"expiration_timeout":    "30m",
							"queue_size":            2 * 4 * 1600,
							"detect_sequence_reset": true,
							"max_message_size":      "10KiB",
							"workers":               100,
						}),
					},
				},
			},
		},
	}

	healthyChan := make(chan struct{})
	closeOnce := sync.Once{}
	server := &mock.StubServerV2{
		CheckinV2Impl: func(observed *proto.CheckinObserved) *proto.CheckinExpected {
			unitState, payload := extractStateAndPayload(observed, "input-unit-1")
			if unitState == proto.State_HEALTHY {
				if payload.streamStatusEquals("netflow-netflow.netflow-1e8b33de-d54a-45cd-90da-23ed71c482e2", map[string]interface{}{
					"status": "HEALTHY",
					"error":  "",
				}) {
					closeOnce.Do(func() { close(healthyChan) })
				}
			}

			return &proto.CheckinExpected{
				Units: allStreams,
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
		"-E", "logging.level=info",
	}
	defer func() {
		os.Args = initialOSArgs
	}()

	beatCmd := filebeat.Filebeat()
	beatRunErr := make(chan error)
	go func() {
		defer close(beatRunErr)
		beatRunErr <- beatCmd.Execute()
	}()

	select {
	case <-healthyChan:
	case err := <-beatRunErr:
		t.Fatalf("beat run err: %v", err)
	case <-time.After(waitFor):
		t.Fatalf("timed out waiting for beat to become healthy")
	}

	registry := monitoring.GetNamespace("dataset").GetRegistry().GetRegistry("netflow_integration_test")

	discardedEventsTotalVar, ok := registry.Get("discarded_events_total").(*monitoring.Uint)
	require.True(t, ok)

	receivedEventTotalVar, ok := registry.Get("received_events_total").(*monitoring.Uint)
	require.True(t, ok)

	udpAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:6006")
	require.NoError(t, err)

	conn, err := net.DialUDP("udp", nil, udpAddr)
	require.NoError(t, err)

	data, err := os.ReadFile("testdata/golden/ipfix_cisco.reversed.pcap.golden.json")
	require.NoError(t, err)

	var expectedFlows struct {
		Flows []beat.Event `json:"events,omitempty"`
	}
	err = json.Unmarshal(data, &expectedFlows)
	require.NoError(t, err)

	f, err := pcap.OpenOffline("testdata/pcap/ipfix_cisco.reversed.pcap")
	require.NoError(t, err)
	defer f.Close()

	var totalBytes, totalPackets int
	rateLimit := 10000
	limiter := rate.NewLimiter(rate.Limit(rateLimit), rateLimit)

	packetSource := gopacket.NewPacketSource(f, f.LinkType())
	for pkt := range packetSource.Packets() {

		if totalPackets%rateLimit == 0 {
			err = limiter.WaitN(ctx, rateLimit)
			require.NoError(t, err)
		}

		payloadData := pkt.TransportLayer().LayerPayload()

		n, err := conn.Write(payloadData)
		require.NoError(t, err)

		totalBytes += n
		totalPackets++
	}

	require.Zero(t, discardedEventsTotalVar.Get())

	require.Eventually(t, func() bool {
		return receivedEventTotalVar.Get() == uint64(totalPackets)
	}, waitFor, tick)

	require.Eventually(t, func() bool {
		return HasDataStream(ctx, outputUsername, outputPassword, outputHost, "logs-netflow.log-default") == nil
	}, waitFor, tick)

	require.Eventually(t, func() bool {
		eventsCount, err := DataStreamEventsCount(ctx, outputUsername, outputPassword, outputHost, "logs-netflow.log-default")
		require.NoError(t, err)
		return eventsCount == uint64(len(expectedFlows.Flows))
	}, waitFor, tick)
}

type unitPayload map[string]interface{}

func (u unitPayload) streamStatusEquals(streamID string, expected map[string]interface{}) bool {
	if u == nil {
		return false
	}

	streams, ok := u["streams"].(map[string]interface{})
	if !ok || streams == nil {
		return false
	}

	streamMap, ok := streams[streamID].(map[string]interface{})
	if !ok || streamMap == nil {
		return false
	}

	return reflect.DeepEqual(streamMap, expected)
}

func extractStateAndPayload(observed *proto.CheckinObserved, inputID string) (proto.State, unitPayload) {
	for _, unit := range observed.GetUnits() {
		if unit.Id == inputID {
			return unit.GetState(), unit.Payload.AsMap()
		}
	}

	return -1, nil
}

type DataStream struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type DataStreamResult struct {
	DataStreams []DataStream `json:"data_streams"`
	Error       interface{}  `json:"error"`
}

func HasDataStream(ctx context.Context, username string, password string, url string, name string) error {
	resultBytes, err := request(ctx, http.MethodGet, username, password, fmt.Sprintf("%s/_data_stream/%s", url, name))
	if err != nil {
		return err
	}

	if resultBytes == nil {
		return errors.New("http not found error")
	}

	var results DataStreamResult
	err = json.Unmarshal(resultBytes, &results)
	if err != nil {
		return err
	}

	if results.Error != nil {
		return fmt.Errorf("error %v while checking for data stream %s", results.Error, name)
	}

	if len(results.DataStreams) != 1 {
		return fmt.Errorf(
			"unexpected count %v of data streams returned when looking for %s",
			len(results.DataStreams), name)
	}

	if results.DataStreams[0].Name != name {
		return fmt.Errorf("unexpected data stream %s returned when looking for %s",
			results.DataStreams[0].Name,
			name)
	}

	return nil
}

// CountResults are the results returned from a _search.
type CountResults struct {
	Count uint64 `json:"count"`
}

func DataStreamEventsCount(ctx context.Context, username string, password string, url string, name string) (uint64, error) {
	resultBytes, err := request(ctx, http.MethodGet, username, password, fmt.Sprintf("%s/%s/_count?q=!_ignored:*+AND+!event.message:*", url, name))
	if err != nil {
		return 0, err
	}

	if resultBytes == nil {
		return 0, errors.New("http not found error")
	}

	var results CountResults
	err = json.Unmarshal(resultBytes, &results)
	if err != nil {
		return 0, err
	}
	return results.Count, nil
}

// DeleteResults are the results returned from a _data_stream delete.
type DeleteResults struct {
	Acknowledged bool `json:"acknowledged"`
}

func DeleteDataStream(ctx context.Context, username string, password string, url string, name string) (bool, error) {
	_, err := request(ctx, http.MethodDelete, username, password, fmt.Sprintf("%s/_data_stream/%s", url, name))
	if err != nil {
		return false, err
	}

	return true, nil
}

func request(ctx context.Context, httpMethod string, username string, password string, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, httpMethod, url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(username, password)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	resultBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return resultBytes, nil
}
