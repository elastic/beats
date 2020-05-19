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

package elastic

import (
	"fmt"
	"strings"
)

type Mode int

const (
	// ModeDefault configures the stack module to collect just a small set of
	// basic metrics and index then into the default Metricbeat index.
	ModeDefault Mode = iota

	// ModeStackMonitoring configures the stack module to collect a richer set
	// of metrics needed to drive the Stack Monitoring UI and index them into
	// Stack Monitoring indices (.monitoring-*).
	ModeStackMonitoring
)

// Unpack unmarshals a Mode string to a Mode. This implements
// ucfg.StringUnpacker.
func (m *Mode) Unpack(str string) error {
	switch strings.ToLower(str) {
	case "default":
		*m = ModeDefault
	case "stack-monitoring":
		*m = ModeStackMonitoring
	default:
		return fmt.Errorf("unknown mode: %v", str)
	}

	return nil
}

// ModeConfig defines the structure for the stack module configuration options
// related to the collection+indexing Mode.
type ModeConfig struct {
	XPackEnabled bool `config:"xpack.enabled"` // Deprecated
	Mode         Mode `config:"mode"`
}

// DefaultModeConfig returns the default mode-related configuration for the stack
// module.
func DefaultModeConfig() ModeConfig {
	return ModeConfig{
		XPackEnabled: false,
		Mode:         ModeStackMonitoring,
	}
}

// GetMode returns the correct mode for the config. It takes into account the deprecated
// and new settings.
func (c ModeConfig) GetMode() Mode {
	if c.XPackEnabled || (c.Mode == ModeStackMonitoring) {
		return ModeStackMonitoring
	}

	return ModeDefault
}
