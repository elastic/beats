package management

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/client/mock"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var raw = `
    id: system/metrics-system-default-system
    name: system-1
    revision: 1
    type: system/metrics
    use_output: default
    meta:
      package:
        name: system
        version: 1.17.0
    data_stream:
      namespace: default
    streams:
      - id: system/metrics-system.cpu-default-system
        data_stream:
          dataset: system.cpu
          type: metrics
        metricsets:
          - cpu
        cpu.metrics:
          - percentages
          - normalized_percentages
        period: 10s
      - id: system/metrics-system.diskio-default-system
        data_stream:
          dataset: system.diskio
          type: metrics
        metricsets:
          - diskio
        diskio.include_devices: null
        period: 10s
      - id: system/metrics-system.filesystem-default-system
        data_stream:
          dataset: system.filesystem
          type: metrics
        metricsets:
          - filesystem
        period: 1m
        processors:
          - drop_event.when.regexp:
              system.filesystem.mount_point: ^/(sys|cgroup|proc|dev|etc|host|lib|snap)($|/)
      - id: system/metrics-system.fsstat-default-system
        data_stream:
          dataset: system.fsstat
          type: metrics
        metricsets:
          - fsstat
        period: 1m
        processors:
          - drop_event.when.regexp:
              system.fsstat.mount_point: ^/(sys|cgroup|proc|dev|etc|host|lib|snap)($|/)
      - id: system/metrics-system.load-default-system
        data_stream:
          dataset: system.load
          type: metrics
        metricsets:
          - load 
        period: 10s
      - id: system/metrics-system.memory-default-system
        data_stream:
          dataset: system.memory
          type: metrics
        metricsets:
          - memory
        period: 10s
      - id: system/metrics-system.network-default-system
        data_stream:
          dataset: system.network
          type: metrics
        metricsets:
          - network
        period: 10s
        network.interfaces: null
      - id: system/metrics-system.process-default-system
        data_stream:
          dataset: system.process
          type: metrics
        metricsets:
          - process
        period: 10s
        process.include_top_n.by_cpu: 5
        process.include_top_n.by_memory: 5
        process.cmdline.cache.enabled: true
        process.cgroups.enabled: false
        process.include_cpu_ticks: false
        processes:
          - .*
      - id: system/metrics-system.process.summary-default-system
        data_stream:
          dataset: system.process.summary
          type: metrics
        metricsets:
          - process_summary
        period: 10s
      - id: system/metrics-system.socket_summary-default-system
        data_stream:
          dataset: system.socket_summary
          type: metrics
        metricsets:
          - socket_summary
        period: 10s
      - id: system/metrics-system.uptime-default-system
        data_stream:
          dataset: system.uptime
          type: metrics
        metricsets:
          - uptime
        period: 10s	
`

func TestAgentControl(t *testing.T) {
	rawExpected := proto.UnitExpectedConfig{
		DataStream: &proto.DataStream{
			Namespace: "default",
		},
		Id:       "system/metrics-system-default-system",
		Type:     "system/metrics",
		Name:     "system-1",
		Revision: 1,
		Meta: &proto.Meta{
			Package: &proto.Package{
				Name:    "system",
				Version: "1.17.0",
			},
		},
		Streams: []*proto.Stream{
			{
				Id: "system/metrics-system.filesystem-default-system",
				DataStream: &proto.DataStream{
					Dataset: "system.filesystem",
					Type:    "metrics",
					Source: requireNewStruct(t, map[string]interface{}{
						"metricsets": []interface{}{"filesystem"},
						"period":     "1m",
						"processors": []interface{}{
							map[string]interface{}{
								"drop_event.when.regexp": map[string]interface{}{
									"system.filesystem.mount_point": "^/(sys|cgroup|proc|dev|etc|host|lib|snap)($|/)",
								},
							},
						},
					}),
				},
			},
		},
	}
	unitOneID := mock.NewID()

	token := mock.NewID()

	var mut sync.Mutex

	t.Logf("Creating mock server")
	srv := mock.StubServerV2{
		CheckinV2Impl: func(observed *proto.CheckinObserved) *proto.CheckinExpected {
			mut.Lock()
			defer mut.Unlock()
			if observed.Token == token {
				// initial checkin
				if len(observed.Units) == 0 || observed.Units[0].State == proto.State_STARTING {
					//gotConfig = true
					t.Logf("Got initial checkin, sending config...")
					return &proto.CheckinExpected{
						Units: []*proto.UnitExpected{
							{
								Id:             unitOneID,
								Type:           proto.UnitType_INPUT,
								ConfigStateIdx: 1,
								Config:         &rawExpected,
								State:          proto.State_HEALTHY,
							},
						},
					}
				} else if observed.Units[0].State == proto.State_STOPPED {
					// remove the unit? I think?
					return &proto.CheckinExpected{
						Units: nil,
					}
				} else if observed.Units[0].State == proto.State_FAILED {
					t.Logf("Unit failed with: %#v", observed.Units[0].Message)
					return &proto.CheckinExpected{
						Units: nil,
					}
				}

			}

			return nil
		},
		ActionImpl: func(response *proto.ActionResponse) error {

			return nil
		},
		ActionsChan: make(chan *mock.PerformAction, 100),
	} // end of srv declaration

	require.NoError(t, srv.Start())
	defer srv.Stop()

	// initialize
	reloader := TestReloader{}
	reload.RegisterV2.MustRegisterList("input", reloader)

	t.Logf("creating client")
	// connect with client
	client := client.NewV2(fmt.Sprintf(":%d", srv.Port), token, client.VersionInfo{
		Name:    "program",
		Version: "v1.0.0",
		Meta: map[string]string{
			"key": "value",
		},
	}, grpc.WithTransportCredentials(insecure.NewCredentials()))

	t.Logf("starting beats client")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	testConfig := Config{Enabled: true}
	mgr, err := NewV2AgentManagerWithClient(&testConfig, reload.RegisterV2, client)
	assert.NoError(t, err)

	err = mgr.Start()
	require.NoError(t, err)
	for {
		select {
		case <-ctx.Done():
			return
		}
	}

}

// TestReloader is a little test interface so we can register a reloader for the V2 config
type TestReloader struct {
}

func (tr TestReloader) Reload(configs []*reload.ConfigWithMeta) error {
	for _, cfg := range configs {
		fmt.Printf("Got config: \n\t%#v\n", cfg.Config)
	}
	return nil
}
