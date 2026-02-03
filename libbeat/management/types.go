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

package management

import "context"

// Copy of github.com/elastic/elastic-agent-client/v7/pkg/proto.AgentInfo to avoid the ELv2 license.
// https://github.com/elastic/elastic-agent-client/blob/112583e0a933bebd719f48d78934b027d884b2b0/elastic-agent-client.proto#L203-L220
type AgentInfo struct {
	ID           string
	Version      string
	Snapshot     bool
	ManagedMode  AgentManagedMode
	Unprivileged bool
}

// Copy of Copy of github.com/elastic/elastic-agent-client/v7/pkg/proto.AgentManagedMode to avoid the ELv2 License
// https://github.com/elastic/elastic-agent-client/blob/112583e0a933bebd719f48d78934b027d884b2b0/elastic-agent-client.proto#L110-L114
type AgentManagedMode int

const (
	AgentManagedMode_MANAGED AgentManagedMode = iota
	AgentManagedMode_STANDALONE
)

// Copy of github.com/elastic/elastic-agent-client/v7/pkg/client.Action to avoid the ELv2 License
// https://github.com/elastic/elastic-agent-client/blob/112583e0a933bebd719f48d78934b027d884b2b0/pkg/client/client.go#L21-L28
type Action interface {
	// Name of the action.
	Name() string

	// Execute performs the action.
	Execute(context.Context, map[string]interface{}) (map[string]interface{}, error)
}

// Copy of github.com/elastic/elastic-agent-client/v7/pkg/client.DiagnosticHook to avoid the ELv2 License.
// https://github.com/elastic/elastic-agent-client/blob/112583e0a933bebd719f48d78934b027d884b2b0/pkg/client/diagnostics.go#L7-L8
type DiagnosticHook func() []byte
