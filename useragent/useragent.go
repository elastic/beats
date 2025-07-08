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

package useragent

import (
	"errors"
	"runtime"
	"strings"
)

type AgentManagementMode int

const (
	// AgentManagementModeManaged indicates that the beat is managed by Fleet.
	AgentManagementModeManaged AgentManagementMode = iota
	// AgentManagementModeStandalone indicates that the beat is running in standalone mode.
	AgentManagementModeStandalone
)

func (m AgentManagementMode) String() string {
	switch m {
	case AgentManagementModeManaged:
		return "Managed"
	case AgentManagementModeStandalone:
		return "Standalone"
	default:
		return "Unknown"
	}
}

// AgentUnprivilegedMode indicates whether the beat is running in unprivileged mode.
type AgentUnprivilegedMode bool

const (
	// AgentUnprivilegedModeUnprivileged indicates that the beat is running in unprivileged mode.
	AgentUnprivilegedModeUnprivileged AgentUnprivilegedMode = true
	// AgentUnprivilegedModePrivileged indicates that the beat is running in privileged mode.
	AgentUnprivilegedModePrivileged AgentUnprivilegedMode = false
)

func (m AgentUnprivilegedMode) String() string {
	if m {
		return "Unprivileged"
	}
	return "Privileged"
}

// UserAgent takes the capitalized name of the current beat and returns
// an RFC compliant user agent string for that beat.
func UserAgent(binaryNameCapitalized string, version, commit, buildTime string, additionalComments ...string) string {
	var builder strings.Builder
	builder.WriteString("Elastic-" + binaryNameCapitalized + "/" + version + " ")
	uaValues := []string{
		runtime.GOOS,
		runtime.GOARCH,
		commit,
		buildTime,
	}
	for _, val := range additionalComments {
		if val != "" {
			uaValues = append(uaValues, val)
		}
	}
	builder.WriteByte('(')
	builder.WriteString(strings.Join(uaValues, "; "))
	builder.WriteByte(')')
	return builder.String()
}

func UserAgentWithBeatTelemetry(binaryNameCapitalized string, version string, mode AgentManagementMode, unprivileged AgentUnprivilegedMode) (string, error) {
	var builder strings.Builder
	builder.WriteString("Elastic-" + binaryNameCapitalized + "/" + version + " ")
	uaValues := []string{
		runtime.GOOS,
		runtime.GOARCH,
		mode.String(),
		unprivileged.String(),
	}
	builder.WriteByte('(')
	builder.WriteString(strings.Join(uaValues, "; "))
	builder.WriteByte(')')

	// Ensure the user agent string does not exceed 100 characters
	userAgent := builder.String()
	if len(userAgent) > 100 {
		return userAgent, errors.New("user agent string exceeds 100 characters")
	}
	return userAgent, nil
}
