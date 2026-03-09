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

package integration

import "strings"

// OutputInspector describes operations for inspecting output.
type OutputInspector interface {
	// Inspect the line of the output and adjust the state accordingly.
	Inspect(string)
	// String is the string representation of the inspector.
	String() string
}

// NewOverallInspector creates an inspector that propagates the line
// to the list of other inspectors.
func NewOverallInspector(inspectors []OutputInspector) OutputInspector {
	return &overallInspector{
		inspectors: inspectors,
	}
}

type overallInspector struct {
	inspectors []OutputInspector
}

func (w *overallInspector) Inspect(line string) {
	for _, inspector := range w.inspectors {
		inspector.Inspect(line)
	}
}

func (w *overallInspector) String() string {
	if len(w.inspectors) == 0 {
		return ""
	}
	inspectors := make([]string, 0, len(w.inspectors))
	for _, inspector := range w.inspectors {
		inspectors = append(inspectors, inspector.String())
	}
	return " * " + strings.Join(inspectors, "\n * ")
}
