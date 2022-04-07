// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build linux
// +build linux

package users

import (
	"net"
	"strconv"

	"github.com/godbus/dbus"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v8/metricbeat/mb"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("system", "users", New)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	counter int
	conn    *dbus.Conn
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The system users metricset is beta.")

	conn, err := initDbusConnection()
	if err != nil {
		return nil, errors.Wrap(err, "error connecting to dbus")
	}

	return &MetricSet{
		BaseMetricSet: base,
		counter:       1,
		conn:          conn,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	sessions, err := listSessions(m.conn)
	if err != nil {
		return errors.Wrap(err, "error listing sessions")
	}

	eventMapping(m.conn, sessions, report)

	return nil
}

// eventMapping iterates through the lists of users and sessions, combining the two
func eventMapping(conn *dbus.Conn, sessions []loginSession, report mb.ReporterV2) error {

	for _, session := range sessions {

		props, err := getSessionProps(conn, session.Path)
		if err != nil {
			return errors.Wrap(err, "error getting properties")
		}

		event := common.MapStr{
			"id":      session.ID,
			"seat":    session.Seat,
			"path":    session.Path,
			"type":    props.Type,
			"service": props.Service,
			"remote":  props.Remote,
			"state":   props.State,
			"scope":   props.Scope,
			"leader":  props.Leader,
		}

		rootEvents := common.MapStr{
			"process": common.MapStr{
				"pid": props.Leader,
			},
			"user": common.MapStr{
				"name": session.User,
				"id":   strconv.Itoa(int(session.UID)),
			},
		}

		if props.Remote {
			event["remote_host"] = props.RemoteHost
			if ipAddr := net.ParseIP(props.RemoteHost); ipAddr != nil {
				rootEvents["source"] = common.MapStr{
					"ip": ipAddr,
				}
			}
		}

		reported := report.Event(mb.Event{
			RootFields:      rootEvents,
			MetricSetFields: event,
		})
		//if the channel is closed and metricbeat is shutting down, just return
		if !reported {
			break
		}
	}
	return nil
}
