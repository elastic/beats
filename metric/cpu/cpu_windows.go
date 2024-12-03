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

//go:build windows

package cpu

import (
	"fmt"

	"github.com/elastic/elastic-agent-libs/helpers/windows/pdh"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/resolve"
)

/*
The below code implements a "metrics tracker" that gives us the ability to
calculate CPU percentages, as we average usage across a time period.
*/

// Monitor is used to monitor the overall CPU usage of the system over time.
type Monitor struct {
	lastSample CPUMetrics
	Hostfs     resolve.Resolver

	// windows specific fields
	query *pdh.Query
}

// New returns a new CPU metrics monitor
// Hostfs is only relevant on linux and freebsd.
func New(hostfs resolve.Resolver, opts ...OptionFunc) (*Monitor, error) {
	var query *pdh.Query
	var err error

	op := option{}
	for _, o := range opts {
		o(&op)
	}
	if !op.usePerformanceCounter {
		if query, err = buildQuery(); err != nil {
			return nil, err
		}
	}

	return &Monitor{
		Hostfs: hostfs,
		query:  query,
	}, nil
}

func buildQuery() (*pdh.Query, error) {
	var q pdh.Query
	if err := q.Open(); err != nil {
		return nil, fmt.Errorf("failed to open query: %w", err)
	}
	// TODO: implement performance counters as a follow up
	return &q, nil
}
