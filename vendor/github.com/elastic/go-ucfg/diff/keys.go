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

package diff

import (
	"fmt"
	"strings"

	ucfg "github.com/elastic/go-ucfg"
)

// Type custom type to identify what was added, remove or keep in the configuration
type Type int

const (
	// Remove keys no longer present in the config
	Remove Type = iota

	// Add keys added from the first config
	Add

	// Keep keys present in both config
	Keep
)

func (dt Type) String() string {
	return []string{
		"-",
		"+",
		" ",
	}[dt]
}

// Diff format of a diff
type Diff map[Type][]string

// String return a human friendly format of the diff
func (d Diff) String() string {
	var lines []string

	for k, values := range d {
		for _, v := range values {
			lines = append(lines, fmt.Sprintf("%s | key: %s", k, v))
		}
	}

	return strings.Join(lines, "\n")
}

// HasChanged returns true if we have remove of added new elements in the graph
func (d *Diff) HasChanged() bool {
	if d.HasKeyAdded() || d.HasKeyRemoved() {
		return true
	}
	return false
}

// HasKeyRemoved returns true if not all keys are present in both configuration
func (d *Diff) HasKeyRemoved() bool {
	if len((*d)[Remove]) > 0 {
		return true
	}

	return false
}

// HasKeyAdded returns true if key were added in the new configuration
func (d *Diff) HasKeyAdded() bool {
	if len((*d)[Add]) > 0 {
		return true
	}
	return false
}

// GoStringer implement the GoStringer interface
func (d Diff) GoStringer() string {
	return d.String()
}

// CompareConfigs takes two configuration and return the difference between the defined keys
func CompareConfigs(old, new *ucfg.Config, opts ...ucfg.Option) Diff {
	oldKeys := old.FlattenedKeys(opts...)
	newKeys := new.FlattenedKeys(opts...)

	difference := make(map[string]Type)

	// Map for candidates check
	for _, k := range oldKeys {
		difference[k] = Remove
	}

	for _, nk := range newKeys {
		if _, ok := difference[nk]; ok {
			difference[nk] = Keep
		} else {
			difference[nk] = Add
		}
	}

	invert := make(Diff)

	for k, v := range difference {
		invert[v] = append(invert[v], k)
	}

	return invert
}
