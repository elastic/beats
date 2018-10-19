// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux

package sockets

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/auditbeat/cache"

	"github.com/elastic/beats/libbeat/logp"
	mbSocket "github.com/elastic/beats/metricbeat/module/system/socket"
)

const (
	moduleName    = "system"
	metricsetName = "sockets"
)

func init() {
	mb.Registry.MustAddMetricSet(moduleName, metricsetName, New,
		mb.DefaultMetricSet(),
	)
}

// MetricSet collects data about sockets.
type MetricSet struct {
	mb.BaseMetricSet
	config Config
	cache  *cache.Cache
	log    *logp.Logger

	netlink *mbSocket.NetlinkSession
}

// New constructs a new MetricSet.
// TODO: Extend beyond Linux.
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

		netlink: mbSocket.NewNetlinkSession(),
	}

	if config.ReportChanges {
		ms.cache = cache.New()
	}

	return ms, nil
}

// Fetch checks which sockets exist on the host and reports them.
// It is invoked periodically.
func (ms *MetricSet) Fetch(report mb.ReporterV2) {
	sockets, err := ms.netlink.GetSocketList()
	if err != nil {
		ms.log.Error(err)
		report.Error(err)
		return
	}
	ms.log.Debugf("netlink returned %d sockets", len(sockets))

	conns := make([]*mbSocket.Connection, 0, len(sockets))
	for _, s := range sockets {
		c := mbSocket.NewConnection(s)
		conns = append(conns, c)
	}

	// Report all current network connections
	var connInfos []common.MapStr

	for _, connInfo := range conns {
		connInfoMapStr := toMapStr(connInfo)

		connInfos = append(connInfos, connInfoMapStr)
	}

	report.Event(mb.Event{
		MetricSetFields: common.MapStr{
			"connection": connInfos,
		},
	})
}

func toMapStr(c *mbSocket.Connection) common.MapStr {
	return common.MapStr{
		"family":      c.Family.String(),
		"state":       c.State.String(),
		"local.ip":    c.LocalIP,
		"local.port":  c.LocalPort,
		"remote.ip":   c.RemoteIP,
		"remote.port": c.RemotePort,
		"inode":       c.Inode,
		"uid":         c.UID,
		"pid":         c.PID,
	}
}
