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

package service

import (
	"path/filepath"

	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

// Config stores the config object
type Config struct {
	StateFilter   []string `config:"service.state_filter"`
	PatternFilter []string `config:"service.pattern_filter"`
}

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("system", "service", New)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	conn     *dbus.Conn
	cfg      Config
	unitList unitFetcher
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The system service metricset is beta.")

	var config Config
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	conn, err := dbus.New()
	if err != nil {
		return nil, errors.Wrap(err, "error connecting to dbus")
	}

	unitFunction, err := instrospectForUnitMethods()
	if err != nil {
		return nil, errors.Wrap(err, "error finding ListUnits Method")
	}

	return &MetricSet{
		BaseMetricSet: base,
		conn:          conn,
		cfg:           config,
		unitList:      unitFunction,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {

	units, err := m.unitList(m.conn, m.cfg.StateFilter, m.cfg.PatternFilter)
	if err != nil {
		return errors.Wrap(err, "error getting list of running units")
	}

	for _, unit := range units {
		//Skip what are basically errors dude to systemd's declarative dependency system
		if unit.LoadState == "not-found" {
			continue
		}

		match, err := filepath.Match("*.service", unit.Name)
		if err != nil {
			m.Logger().Errorf("Error matching unit service %s: %s", unit.Name, err)
			continue
		}
		// If we don't have a *.service, skip
		if !match {
			continue
		}

		props, err := getProps(m.conn, unit.Name)
		if err != nil {
			m.Logger().Errorf("error getting properties for service: %s", err)
			continue
		}

		event, err := formProperties(unit, props)
		if err != nil {
			m.Logger().Errorf("Error getting properties for systemd service %s: %s", unit.Name, err)
			continue
		}

		isOpen := report.Event(event)
		if !isOpen {
			return nil
		}

	}
	return nil
}

// Get Properties for a given unit, cast to a struct
func getProps(conn *dbus.Conn, unit string) (Properties, error) {
	rawProps, err := conn.GetAllProperties(unit)
	if err != nil {
		return Properties{}, errors.Wrap(err, "error getting list of running units")
	}
	parsed := Properties{}
	if err := mapstructure.Decode(rawProps, &parsed); err != nil {
		return parsed, errors.Wrap(err, "error decoding properties")
	}
	return parsed, nil
}
