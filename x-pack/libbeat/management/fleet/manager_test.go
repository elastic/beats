// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fleet

import (
	"testing"

	"github.com/elastic/beats/v7/libbeat/common"

	"github.com/elastic/beats/v7/libbeat/common/reload"
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

type dummyReloadable struct{}

func (dummyReloadable) Reload(config *reload.ConfigWithMeta) error {
	return nil
}
