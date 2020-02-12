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

package adapter

import (
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/monitoring"
)

// provide filters for filtering and adapting a metric type
// to monitoring.Var.

// MetricFilter type used to defined and combine filters.
type MetricFilter func(*metricFilters) *metricFilters

// metricFilters provides set of filters to apply to a new metric.
type metricFilters struct {
	filters []varFilter
}

type varFilter func(state) state

// state provides the filter state to be changed by every filter.
//
// After filtering the state will be used to choose on the metric name, the
// metric type , or wether the metric is to be ignored.
type state struct {
	kind   kind
	action action
	mode   monitoring.Mode
	reg    *monitoring.Registry
	name   string
	metric interface{}
}

// action defines the action to be
type action uint8

// kind defines the kind of operation to be executed
type kind uint8

const (
	kndFind kind = iota
	kndAdd
	kndRemove
)

const (
	actIgnore action = iota
	actAccept
)

func makeFilters(in ...MetricFilter) *metricFilters {
	if len(in) == 0 {
		return nil
	}

	m := &metricFilters{}
	for _, mk := range in {
		m = mk(m)
	}
	return m
}

func (m *metricFilters) apply(st state) state {
	if m != nil {
		for _, filter := range m.filters {
			st = filter(st)
		}
	}
	return st
}

func ApplyIf(pred func(name string) bool, filters ...MetricFilter) MetricFilter {
	then := makeFilters(filters...)
	return withVarFilter(func(st state) state {
		if pred(st.name) {
			st = then.apply(st)
		}
		return st
	})
}

var Accept = withVarFilter(func(st state) state {
	st.action = actAccept
	return st
})

// WhitelistIf will accept a metric if the metrics name matches
// the given predicate.
func WhitelistIf(pred func(string) bool) MetricFilter {
	return ApplyIf(pred, Accept)
}

// NameIn checks a metrics name matching any of the set names
func NameIn(names ...string) func(string) bool {
	return common.MakeStringSet(names...).Has
}

// Whitelist sets a list of metric names to be accepted.
func Whitelist(names ...string) MetricFilter {
	return WhitelistIf(NameIn(names...))
}

// ReportIf sets variable report mode for all metrics satisfying the predicate.
func ReportIf(pred func(string) bool) MetricFilter {
	return ApplyIf(pred, withVarFilter(func(st state) state {
		st.mode = monitoring.Reported
		return st
	}))
}

// ReportNames enables reporting for all metrics matching any of the given names.
func ReportNames(names ...string) MetricFilter {
	return ReportIf(NameIn(names...))
}

// ModifyName changes a metric its name using the provided
// function.
func ModifyName(f func(string) string) MetricFilter {
	return withVarFilter(func(st state) state {
		st.name = f(st.name)
		return st
	})
}

// Rename renames a metric to `to`, if the names matches `from`
// If the name matches, it will be automatically white-listed.
func Rename(from, to string) MetricFilter {
	return withVarFilter(func(st state) state {
		if st.name == from {
			st.action, st.name = actAccept, to
		}
		return st
	})
}

// NameReplace replaces substrings in a metrics names with `new`.
func NameReplace(old, new string) MetricFilter {
	return ModifyName(func(name string) string {
		return strings.Replace(name, old, new, -1)
	})
}

// ToLowerName converts all metric names to lower-case
var ToLowerName = ModifyName(strings.ToLower)

// ToUpperName converts all metric name to upper-case
var ToUpperName = ModifyName(strings.ToUpper)

// withVarFilter lifts a varFilter into a MetricFilter
func withVarFilter(f varFilter) MetricFilter {
	return func(m *metricFilters) *metricFilters {
		m.filters = append(m.filters, f)
		return m
	}
}
