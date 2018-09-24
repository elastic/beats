// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package host

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/auditbeat/module/system/config"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/go-sysinfo"
)

const (
	moduleName    = "system"
	metricsetName = "host"
)

func init() {
	mb.Registry.MustAddMetricSet(moduleName, metricsetName, New,
		mb.DefaultMetricSet(),
	)
}

// MetricSet collects data about the host.
type MetricSet struct {
	mb.BaseMetricSet
	log *logp.Logger
}

// New constructs a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Experimental("The %v/%v dataset is experimental", moduleName, metricsetName)

	config := config.NewDefaultConfig()
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, errors.Wrapf(err, "failed to unpack the %v/%v config", moduleName, metricsetName)
	}

	ms := &MetricSet{
		BaseMetricSet: base,
		log:           logp.NewLogger(moduleName),
	}

	if config.ReportChanges {
		// TODO: Implement reporting changes?
		ms.log.Warnw("Metricset %v/%v does not support report_changes", moduleName, metricsetName)
	}

	return ms, nil
}

// Fetch collects data about the host. It is invoked periodically.
func (ms *MetricSet) Fetch(report mb.ReporterV2) {
	host, err := sysinfo.Host()
	if err != nil {
		report.Error(errors.Wrap(err, "Failed to load host information"))
		return
	}

	report.Event(mb.Event{
		MetricSetFields: common.MapStr{
			// https://github.com/elastic/ecs#-host-fields
			"uptime":              host.Info().Uptime(),
			"boottime":            host.Info().BootTime,
			"containerized":       host.Info().Containerized,
			"timezone.name":       host.Info().Timezone,
			"timezone.offset.sec": host.Info().TimezoneOffsetSec,
			"name":                host.Info().Hostname,
			"id":                  host.Info().UniqueID,
			"ip":                  host.Info().IPs,
			"mac":                 host.Info().MACs,
			// TODO "host.type": ?
			"architecture": host.Info().Architecture,

			// https://github.com/elastic/ecs#-operating-system-fields
			"os": common.MapStr{
				"platform": host.Info().OS.Platform,
				"name":     host.Info().OS.Name,
				"family":   host.Info().OS.Family,
				"version":  host.Info().OS.Version,
				"kernel":   host.Info().KernelVersion,
			},
		},
	})
}
