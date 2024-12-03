// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package process

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/elastic/beats/v7/auditbeat/ab"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system"
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	metricsetName = "process"
	namespace     = "system.audit.process"

	eventTypeState = "state"
	eventTypeEvent = "event"
)

// MetricSet collects data about the host.
type MetricSet struct {
	config Config
	log    *logp.Logger
}

type eventAction uint8

const (
	eventActionExistingProcess eventAction = iota
	eventActionProcessStarted
	eventActionProcessStopped
	eventActionProcessError
)

func (action eventAction) String() string {
	switch action {
	case eventActionExistingProcess:
		return "existing_process"
	case eventActionProcessStarted:
		return "process_started"
	case eventActionProcessStopped:
		return "process_stopped"
	case eventActionProcessError:
		return "process_error"
	default:
		return ""
	}
}

func (action eventAction) Type() string {
	switch action {
	case eventActionExistingProcess:
		return "info"
	case eventActionProcessStarted:
		return "start"
	case eventActionProcessStopped:
		return "end"
	case eventActionProcessError:
		return "info"
	default:
		return "info"
	}
}

func init() {
	ab.Registry.MustAddMetricSet(system.ModuleName, metricsetName, New,
		mb.DefaultMetricSet(),
		mb.WithNamespace(namespace),
	)
}

// New constructs a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	var ms MetricSet

	cfgwarn.Beta("The %v/%v dataset is beta", system.ModuleName, metricsetName)

	ms.config = defaultConfig
	ms.log = logp.NewLogger(metricsetName)

	if err := base.Module().UnpackConfig(&ms.config); err != nil {
		return nil, fmt.Errorf("failed to unpack the %v/%v config: %w", system.ModuleName, metricsetName, err)
	}

	return NewFromSysInfo(base, ms)
}

// entityID creates an ID that uniquely identifies this process across machines.
func entityID(hostID string, pid int, startTime time.Time) string {
	h := system.NewEntityHash()
	h.Write([]byte(hostID))
	//nolint:errcheck // no error handling
	binary.Write(h, binary.LittleEndian, int64(pid))
	//nolint:errcheck // no error handling
	binary.Write(h, binary.LittleEndian, int64(startTime.Nanosecond()))
	return h.Sum()
}
