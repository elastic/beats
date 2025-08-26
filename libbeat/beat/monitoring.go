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

package beat

import "github.com/elastic/elastic-agent-libs/monitoring"

type Monitoring struct {
	// Previously monitoring.GetNamespace("info")
	infoRegistry *monitoring.Registry

	// Previously monitoring.GetNamespace("state")
	stateRegistry *monitoring.Registry

	// Previously monitoring.Default or monitoring.GetNamespace("stats")
	statsRegistry *monitoring.Registry

	// Previously monitoring.GetNamespace("dataset")
	inputsRegistry *monitoring.Registry
}

// Returns a Monitoring struct that shadows the legacy global monitoring API,
// which can be used within standalone beats to guarantee full interoperability
// with components that are not yet migrated to report metrics via a
// beat.Monitoring field.
func NewGlobalMonitoring() Monitoring {
	return Monitoring{
		statsRegistry: monitoring.Default,
		stateRegistry: monitoring.GetNamespace("state").GetRegistry(),
		infoRegistry:  monitoring.GetNamespace("info").GetRegistry(),

		inputsRegistry: monitoring.GetNamespace("dataset").GetRegistry(),
	}
}

// Returns a new initialized Monitoring struct for use in a Beats Receiver
// or in unit tests. Will not reflect metrics created through the legacy
// global API (monitoring.Default, monitoring.GetNamespace, etc); for
// full interoperability with the global API, use NewGlobalMonitoring.
func NewMonitoring() Monitoring {
	return Monitoring{
		statsRegistry: monitoring.NewRegistry(),
		stateRegistry: monitoring.NewRegistry(),
		infoRegistry:  monitoring.NewRegistry(),

		inputsRegistry: monitoring.NewRegistry(),
	}
}

// The top-level info registry for the Beat or Beat Receiver, formerly accessed
// via monitoring.GetNamespace("info").
func (m Monitoring) InfoRegistry() *monitoring.Registry {
	return m.infoRegistry
}

// The top-level state registry for the Beat or Beat Receiver, formerly accessed
// via monitoring.GetNamespace("state").
func (m Monitoring) StateRegistry() *monitoring.Registry {
	return m.stateRegistry
}

// The top-level stats / "default" registry for the Beat or Beat Receiver,
// formerly accessed via monitoring.GetNamespace("stats") or
// monitoring.Default. Published in internal monitoring as "metrics" for
// compatibility with other components.
func (m Monitoring) StatsRegistry() *monitoring.Registry {
	return m.statsRegistry
}

// The top-level input metrics registry for the Beat or Beat Receiver, formerly
// accessed via monitoring.GetNamespace("dataset").
func (m Monitoring) InputsRegistry() *monitoring.Registry {
	return m.inputsRegistry
}
