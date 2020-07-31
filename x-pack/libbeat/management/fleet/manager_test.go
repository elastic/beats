// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fleet

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-client/v7/pkg/proto"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/libbeat/management"
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

func TestStatusToProtoStatus(t *testing.T) {
	assert.Equal(t, proto.StateObserved_HEALTHY, statusToProtoStatus(management.Unknown))
	assert.Equal(t, proto.StateObserved_STARTING, statusToProtoStatus(management.Starting))
	assert.Equal(t, proto.StateObserved_CONFIGURING, statusToProtoStatus(management.Configuring))
	assert.Equal(t, proto.StateObserved_HEALTHY, statusToProtoStatus(management.Running))
	assert.Equal(t, proto.StateObserved_DEGRADED, statusToProtoStatus(management.Degraded))
	assert.Equal(t, proto.StateObserved_FAILED, statusToProtoStatus(management.Failed))
	assert.Equal(t, proto.StateObserved_STOPPING, statusToProtoStatus(management.Stopping))
}

type dummyReloadable struct{}

func (dummyReloadable) Reload(config *reload.ConfigWithMeta) error {
	return nil
}
