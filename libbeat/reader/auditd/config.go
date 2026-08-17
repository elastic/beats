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

package auditd

import "fmt"

// Mode controls what the auditd parser does with each log line.
type Mode string

const (
	// ModeNone disables auditd parsing entirely (pass-through).
	ModeNone Mode = "none"
	// ModeParse parses each line independently, populating auditd.log.* fields.
	ModeParse Mode = "parse"
	// ModeCoalesce groups related records by sequence number and produces
	// compound events in the auditd.data.* namespace using aucoalesce.
	ModeCoalesce Mode = "coalesce"
)

// Unpack implements the config.Unpacker interface for Mode.
func (m *Mode) Unpack(s string) error {
	switch Mode(s) {
	case ModeNone, ModeParse, ModeCoalesce:
		*m = Mode(s)
		return nil
	default:
		return fmt.Errorf("invalid auditd parser mode %q (valid: none, parse, coalesce)", s)
	}
}

// Config stores the configuration for the auditd parser.
type Config struct {
	// Mode controls the parser behaviour. Default is "parse" (per-line parsing).
	Mode Mode `config:"mode"`
	// LogErrors, if true, logs parse errors via the parser's logger.
	LogErrors bool `config:"log_errors"`
	// AddErrorKey, if true, adds a parse error to the event under error.message.
	AddErrorKey bool `config:"add_error_key"`
}

// DefaultConfig returns a Config populated with default values. The default
// mode is "parse" for backward compatibility: the auditd parser section is only
// present when the integration has opted in via the use_auditd_parser toggle,
// so its mere presence implies that parsing is desired.
func DefaultConfig() Config {
	return Config{
		Mode:        ModeParse,
		LogErrors:   false,
		AddErrorKey: true,
	}
}
