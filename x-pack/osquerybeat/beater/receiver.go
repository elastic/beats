// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
)

// NewReceiver creates an instance of osquerybeat for use as an OTel receiver.
// It disables orphan detection since the receiver lifecycle is managed by the
// OTel collector, not a parent process.
func NewReceiver(b *beat.Beat, cfg *conf.C) (beat.Beater, error) {
	bt, err := New(b, cfg)
	if err != nil {
		return nil, err
	}
	ob, ok := bt.(*osquerybeat)
	if !ok {
		return nil, fmt.Errorf("unexpected beater type %T", bt)
	}
	ob.disableWatcher = true
	return ob, nil
}
