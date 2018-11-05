// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package login

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/auditbeat/cache"

	"github.com/elastic/beats/libbeat/logp"
)

const (
	moduleName    = "system"
	metricsetName = "login"
)

func init() {
	mb.Registry.MustAddMetricSet(moduleName, metricsetName, New,
		mb.DefaultMetricSet(),
	)
}

// MetricSet collects login records from /var/log/wtmp.
type MetricSet struct {
	mb.BaseMetricSet
	config   Config
	osFamily string
	cache    *cache.Cache
	log      *logp.Logger
}

// New constructs a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Experimental("The %v/%v dataset is experimental", moduleName, metricsetName)

	config := defaultConfig
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, errors.Wrapf(err, "failed to unpack the %v/%v config", moduleName, metricsetName)
	}

	ms := &MetricSet{
		BaseMetricSet: base,
		config:        config,
		log:           logp.NewLogger(moduleName),
	}

	return ms, nil
}

// Fetch collects any new login records from /var/log/wtmp. It is invoked periodically.
func (ms *MetricSet) Fetch(report mb.ReporterV2) {
	loginRecords, err := ReadUtmpFile(ms.config.WtmpFile)
	if err != nil {
		ms.log.Error(err)
		report.Error(err)
		return
	}

	for _, loginRecord := range loginRecords {
		report.Event(mb.Event{
			MetricSetFields: common.MapStr{
				"login": loginRecord.toMapStr(),
			},
		})
	}
}

/*
func convertToCacheable(packages []*Package) []cache.Cacheable {
	c := make([]cache.Cacheable, 0, len(packages))

	for _, p := range packages {
		c = append(c, p)
	}

	return c
}
*/
