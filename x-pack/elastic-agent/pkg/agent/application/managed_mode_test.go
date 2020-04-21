// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/filters"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
)

func TestManagedModeRouting(t *testing.T) {
	streams := make(map[routingKey]stream)
	streamFn := func(l *logger.Logger, r routingKey) (stream, error) {
		m := newMockStreamStore()
		streams[r] = m

		return m, nil
	}

	log, _ := logger.New()
	router, _ := newRouter(log, streamFn)
	emit := emitter(log, router, &configModifiers{Decorators: []decoratorFunc{injectMonitoring}, Filters: []filterFunc{filters.ConstraintFilter}})

	actionDispatcher, err := newActionDispatcher(context.Background(), log, &handlerDefault{log: log})
	assert.NoError(t, err)

	actionDispatcher.MustRegister(
		&fleetapi.ActionConfigChange{},
		&handlerConfigChange{
			log:     log,
			emitter: emit,
		},
	)

	actions, err := testActions()
	assert.NoError(t, err)

	err = actionDispatcher.Dispatch(newNoopAcker(), actions...)
	assert.NoError(t, err)

	// has 1 config request for fb, mb and monitoring?
	assert.Equal(t, 1, len(streams))

	defaultStreamStore, found := streams["default"]
	assert.True(t, found, "default group not found")
	assert.Equal(t, 1, len(defaultStreamStore.(*mockStreamStore).store))

	confReq := defaultStreamStore.(*mockStreamStore).store[0]
	assert.Equal(t, 3, len(confReq.ProgramNames()))
	assert.Equal(t, monitoringName, confReq.ProgramNames()[2])
}

func testActions() ([]action, error) {
	checkinResponse := &fleetapi.CheckinResponse{}
	if err := json.Unmarshal([]byte(fleetResponse), &checkinResponse); err != nil {
		return nil, err
	}

	return checkinResponse.Actions, nil
}

type mockStreamStore struct {
	store []*configRequest
}

func newMockStreamStore() *mockStreamStore {
	return &mockStreamStore{
		store: make([]*configRequest, 0),
	}
}

func (m *mockStreamStore) Execute(cr *configRequest) error {
	m.store = append(m.store, cr)
	return nil
}

func (m *mockStreamStore) Close() error {
	return nil
}

const fleetResponse = `
{
  "action": "checkin",
  "success": true,
  "actions": [
    {
      "agent_id": "17e93530-7f42-11ea-9330-71e968b29fa4",
      "type": "CONFIG_CHANGE",
      "data": {
        "config": {
          "id": "86561d50-7f3b-11ea-9fab-3db3bdb4efa4",
          "outputs": {
            "default": {
              "type": "elasticsearch",
              "hosts": [
                "http://localhost:9200"
              ],
              "api_key": "pNr6fnEBupQ3-5oEEkWJ:FzhrQOzZSG-Vpsq9CGk4oA"
            }
          },
          "datasources": [
            {
              "id": "system-1",
              "enabled": true,
              "use_output": "default",
              "inputs": [
                {
                  "type": "system/metrics",
                  "enabled": true,
                  "streams": [
                    {
                      "id": "system/metrics-system.core",
                      "enabled": true,
                      "dataset": "system.core",
                      "period": "10s",
                      "metrics": [
                        "percentages"
                      ]
                    },
                    {
                      "id": "system/metrics-system.cpu",
                      "enabled": true,
                      "dataset": "system.cpu",
                      "period": "10s",
                      "metrics": [
                        "percentages",
                        "normalized_percentages"
                      ]
                    },
                    {
                      "id": "system/metrics-system.diskio",
                      "enabled": true,
                      "dataset": "system.diskio",
                      "period": "10s",
                      "include_devices": []
                    },
                    {
                      "id": "system/metrics-system.entropy",
                      "enabled": true,
                      "dataset": "system.entropy",
                      "period": "10s",
                      "include_devices": []
                    },
                    {
                      "id": "system/metrics-system.filesystem",
                      "enabled": true,
                      "dataset": "system.filesystem",
                      "period": "1m",
                      "ignore_types": []
                    },
                    {
                      "id": "system/metrics-system.fsstat",
                      "enabled": true,
                      "dataset": "system.fsstat",
                      "period": "1m",
                      "ignore_types": []
                    },
                    {
                      "id": "system/metrics-system.load",
                      "enabled": true,
                      "dataset": "system.load",
                      "period": "10s"
                    },
                    {
                      "id": "system/metrics-system.memory",
                      "enabled": true,
                      "dataset": "system.memory",
                      "period": "10s"
                    },
                    {
                      "id": "system/metrics-system.network",
                      "enabled": true,
                      "dataset": "system.network",
                      "period": "10s"
                    },
                    {
                      "id": "system/metrics-system.network_summary",
                      "enabled": true,
                      "dataset": "system.network_summary",
                      "period": "10s"
                    },
                    {
                      "id": "system/metrics-system.process",
                      "enabled": true,
                      "dataset": "system.process",
                      "period": "10s",
                      "processes": [
                        ".*"
                      ],
                      "include_top_n.enabled": true,
                      "include_top_n.by_cpu": 5,
                      "include_top_n.by_memory": 5,
                      "cmdline.cache.enabled": true,
                      "cgroups.enabled": true,
                      "env.whitelist": [],
                      "include_cpu_ticks": false
                    },
                    {
                      "id": "system/metrics-system.process_summary",
                      "enabled": true,
                      "dataset": "system.process_summary",
                      "period": "10s"
                    },
                    {
                      "id": "system/metrics-system.raid",
                      "enabled": true,
                      "dataset": "system.raid",
                      "period": "10s",
                      "mount_point": "/"
                    },
                    {
                      "id": "system/metrics-system.service",
                      "enabled": true,
                      "dataset": "system.service",
                      "period": "10s",
                      "state_filter": []
                    },
                    {
                      "id": "system/metrics-system.socket_summary",
                      "enabled": true,
                      "dataset": "system.socket_summary",
                      "period": "10s"
                    },
                    {
                      "id": "system/metrics-system.uptime",
                      "enabled": true,
                      "dataset": "system.uptime",
                      "period": "15m"
                    },
                    {
                      "id": "system/metrics-system.users",
                      "enabled": true,
                      "dataset": "system.users",
                      "period": "10s"
                    }
                  ]
                },
                {
                  "type": "logs",
                  "enabled": true,
                  "streams": [
                    {
                      "id": "logs-system.auth",
                      "enabled": true,
                      "dataset": "system.auth",
                      "paths": [
                        "/var/log/auth.log*",
                        "/var/log/secure*"
                      ]
                    },
                    {
                      "id": "logs-system.syslog",
                      "enabled": true,
                      "dataset": "system.syslog",
                      "paths": [
                        "/var/log/messages*",
                        "/var/log/syslog*"
                      ]
                    }
                  ]
                }
              ],
              "package": {
                "name": "system",
                "version": "0.9.0"
              }
            }
          ],
          "revision": 3,
          "settings.monitoring": {
            "use_output": "default",
            "enabled": true,
            "logs": true,
            "metrics": true
          }
        }
      },
      "id": "1c7e26a0-7f42-11ea-9330-71e968b29fa4",
      "created_at": "2020-04-15T17:54:11.081Z"
    }
  ]
}	
	`
