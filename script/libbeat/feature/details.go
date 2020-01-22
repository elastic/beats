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

package feature

import "fmt"

// Describer contains general information for a specific feature, the fields will be used to report
// useful information by the factories or any future CLI.
type Describer interface {
	// Stability is the stability of the Feature, this allow the user to filter embedded functionality
	// by their maturity at runtime.
	// Example: Beta, Experimental, Stable or Undefined.
	Stability() Stability

	// Doc is a one liner describing the current feature.
	// Example: Dissect allows to define patterns to extract useful information from a string.
	Doc() string

	// FullName is the human readable name of the feature.
	// Example: Jolokia Discovery
	FullName() string
}

// Details minimal information that you must provide when creating a feature.
type Details struct {
	stability Stability
	doc       string
	fullName  string
}

// Stability is the stability of the Feature, this allow the user to filter embedded functionality
// by their maturity at runtime.
// Example: Beta, Experimental, Stable or Undefined.
func (d *Details) Stability() Stability {
	return d.stability
}

// Doc is a one liner describing the current feature.
// Example: Dissect allows to define patterns to extract useful information from a string.
func (d *Details) Doc() string {
	return d.doc
}

// FullName is the human readable name of the feature.
// Example: Jolokia Discovery
func (d *Details) FullName() string {
	return d.fullName
}

func (d *Details) String() string {
	return fmt.Sprintf(
		"name: %s, description: %s (stability: %s)",
		d.fullName,
		d.doc,
		d.stability,
	)
}

// NewDetails return the minimal information a new feature must provide.
func NewDetails(fullName string, doc string, stability Stability) *Details {
	return &Details{fullName: fullName, doc: doc, stability: stability}
}
