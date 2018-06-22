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

package monitoring

import "errors"

type Mode uint8

//go:generate stringer -type=Mode
const (
	// Reported mode, is lowest report level with most basic metrics only
	Reported Mode = iota

	// Full reports all metrics
	Full
)

// Default is the global default metrics registry provided by the monitoring package.
var Default = NewRegistry()

func init() {
	GetNamespace("stats").SetRegistry(Default)
}

var errNotFound = errors.New("Name unknown")
var errInvalidName = errors.New("Name does not point to a valid variable")

func VisitMode(mode Mode, vs Visitor) {
	Default.Visit(mode, vs)
}

func Visit(vs Visitor) {
	Default.Visit(Full, vs)
}

func Do(mode Mode, f func(string, interface{})) {
	Default.Do(mode, f)
}

func Get(name string) Var {
	return Default.Get(name)
}

func GetRegistry(name string) *Registry {
	return Default.GetRegistry(name)
}

func Remove(name string) {
	Default.Remove(name)
}

func Clear() error {
	return Default.Clear()
}
