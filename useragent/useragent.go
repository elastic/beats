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
	"runtime"
	"strings"
)

type AgentManagementMode int

const (
	// AgentManagementModeUnknown indicates that the management mode is unknown.
	AgentManagementModeUnknown AgentManagementMode = iota
	// AgentManagementModeStandalone indicates that the beat is not running under agent.
	AgentManagementModeStandalone
	// AgentManagementModeUnmanaged indicates that the beat is running under agent but not managed by Fleet.
	AgentManagementModeUnmanaged
	// AgentManagementModeManaged indicates that the beat is running under agent and managed by Fleet.
	AgentManagementModeManaged
)

func (m AgentManagementMode) String() string {
	switch m {
	case AgentManagementModeUnmanaged:
		return "Unmanaged"
	case AgentManagementModeManaged:
		return "Managed"
	case AgentManagementModeStandalone:
		return "Standalone"
	default:
		return "Unknown"
	}
}

// AgentUnprivilegedMode indicates whether the beat is running in unprivileged mode.
type AgentUnprivilegedMode int8

const (
	// AgentUnprivilegedModeUnknown indicates privilege mode is unknown.
	AgentUnprivilegedModeUnknown AgentUnprivilegedMode = iota
	// AgentUnprivilegedModeUnprivileged indicates that the beat is running in unprivileged mode.
	AgentUnprivilegedModeUnprivileged
	// AgentUnprivilegedModePrivileged indicates that the beat is running in privileged mode.
	AgentUnprivilegedModePrivileged
)

func (m AgentUnprivilegedMode) String() string {
	switch m {
	case AgentUnprivilegedModeUnprivileged:
		return "Unprivileged"
	case AgentUnprivilegedModePrivileged:
		return "Privileged"
	default:
		return "Unknown"
	}
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

func UserAgentWithBeatTelemetry(
	binaryNameCapitalized string,
	version string,
	mode AgentManagementMode,
	unprivileged AgentUnprivilegedMode,
	isFIPSDistribution bool,
) string {
	var builder strings.Builder
	builder.WriteString("Elastic-" + binaryNameCapitalized + "/" + version + " ")
	uaValues := []string{
		runtime.GOOS,
		runtime.GOARCH,
	}
	if mode != AgentManagementModeUnknown {
		uaValues = append(uaValues, mode.String())
	}
	if unprivileged != AgentUnprivilegedModeUnknown {
		uaValues = append(uaValues, unprivileged.String())
	}
	if isFIPSDistribution {
		uaValues = append(uaValues, "FIPS")
	}

	builder.WriteByte('(')
	builder.WriteString(strings.Join(uaValues, "; "))
	builder.WriteByte(')')

	return builder.String()
}
