// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-client/v7/pkg/proto"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/libbeat/logp"
	lbmanagement "github.com/elastic/beats/v7/libbeat/management"
)

func TestConfigBlocks(t *testing.T) {
	input := `
filebeat:
  inputs:
    - type: log
      paths:
        - /var/log/hello1.log
        - /var/log/hello2.log
output:
  elasticsearch:
    hosts:
      - localhost:9200`

	var cfg common.MapStr
	uconfig, err := common.NewConfigFrom(input)
	if err != nil {
		t.Fatalf("Config blocks unsuccessfully generated: %+v", err)
	}

	err = uconfig.Unpack(&cfg)
	if err != nil {
		t.Fatalf("Config blocks unsuccessfully generated: %+v", err)
	}

	reg := reload.NewRegistry()
	reg.Register("output", &dummyReloadable{})
	reg.Register("filebeat.inputs", &dummyReloadable{})

	cm := &Manager{
		registry: reg,
	}
	blocks, err := cm.toConfigBlocks(cfg)
	if err != nil {
		t.Fatalf("Config blocks unsuccessfully generated: %+v", err)
	}

	if len(blocks) != 2 {
		t.Fatalf("Expected 2 block have %d: %+v", len(blocks), blocks)
	}
}

func TestAssertPresenceOfInputsAndOutput(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected bool
	}{
		"return true when output and inputs are present": {
			input: `
filebeat:
  inputs:
    - type: log
      paths:
        - /var/log/hello1.log
        - /var/log/hello2.log

    - type: filestream
      paths:
        - /var/log/hello3.log
        - /var/log/hello4.log
output:
  elasticsearch:
    hosts:
      - localhost:9200`,
			expected: true,
		},
		"return false when output is prevent and inputs are present": {
			input: `
filebeat:
  inputs:
    - type: log
      paths:
        - /var/log/hello1.log
        - /var/log/hello2.log

    - type: filestream
      paths:
        - /var/log/hello3.log
        - /var/log/hello4.log`,
			expected: false,
		},
		"return false when output is present and inputs are 0": {
			input: `
filebeat:
  inputs:
output:
  elasticsearch:
    hosts:
      - localhost:9200`,
			expected: false,
		},
		"return false when output is present and inputs are absent": {
			input: `
filebeat:
output:
  elasticsearch:
    hosts:
      - localhost:9200`,
			expected: false,
		},
		"return false when output is absent and inputs are present": {
			input: `
filebeat:
  inputs:
    - type: log
      paths:
        - /var/log/hello1.log
        - /var/log/hello2.log

    - type: filestream
      paths:
        - /var/log/hello3.log
        - /var/log/hello4.log`,
			expected: false,
		},
		"return false when output is nil and inputs are present": {
			input: `
output:
filebeat:
  inputs:
    - type: log
      paths:
        - /var/log/hello1.log
        - /var/log/hello2.log

    - type: filestream
      paths:
        - /var/log/hello3.log
        - /var/log/hello4.log`,
			expected: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// NOTE(ph): I am not a big fan of copying this over from the Manager.OnConfig function
			// But I want to be as closes of the real transformation and we cannot refactor that code
			// while debugging.
			cm := dummyCM()

			var configMap common.MapStr
			uconfig, err := common.NewConfigFrom(test.input)
			require.NoError(t, err)

			err = uconfig.Unpack(&configMap)
			require.NoError(t, err)

			blocks, err := cm.toConfigBlocks(configMap)
			require.NoError(t, err)

			require.Equal(t, test.expected, assertPresenceOfInputsAndOutput(logp.NewLogger("tests"), blocks))
		})
	}
}

func TestStatusToProtoStatus(t *testing.T) {
	assert.Equal(t, proto.StateObserved_HEALTHY, statusToProtoStatus(lbmanagement.Unknown))
	assert.Equal(t, proto.StateObserved_STARTING, statusToProtoStatus(lbmanagement.Starting))
	assert.Equal(t, proto.StateObserved_CONFIGURING, statusToProtoStatus(lbmanagement.Configuring))
	assert.Equal(t, proto.StateObserved_HEALTHY, statusToProtoStatus(lbmanagement.Running))
	assert.Equal(t, proto.StateObserved_DEGRADED, statusToProtoStatus(lbmanagement.Degraded))
	assert.Equal(t, proto.StateObserved_FAILED, statusToProtoStatus(lbmanagement.Failed))
	assert.Equal(t, proto.StateObserved_STOPPING, statusToProtoStatus(lbmanagement.Stopping))
}

type dummyReloadable struct{}

func (dummyReloadable) Reload(config *reload.ConfigWithMeta) error {
	return nil
}

type dummyReloadableList struct{}

func (dummyReloadableList) ReloadableList() error {
	return nil
}

func dummyCM() *Manager {
	// Use the 3 register point of Filebeat.
	reg := reload.NewRegistry()
	reg.Register("output", &dummyReloadable{})
	reg.Register("filebeat.inputs", &dummyReloadable{})
	reg.Register("filebeat.modules", &dummyReloadable{})

	cm := &Manager{
		registry: reg,
		logger:   logp.NewLogger("test"),
	}

	return cm
}
