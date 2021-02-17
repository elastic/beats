// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/state"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/status"
)

type noopController struct{}

func (*noopController) RegisterComponent(_ string) status.Reporter { return &noopReporter{} }
func (*noopController) RegisterComponentWithPersistance(_ string, _ bool) status.Reporter {
	return &noopReporter{}
}
func (*noopController) RegisterApp(_ string, _ string) status.Reporter { return &noopReporter{} }
func (*noopController) Status() status.AgentStatus                     { return status.AgentStatus{Status: status.Healthy} }
func (*noopController) StatusCode() status.AgentStatusCode             { return status.Healthy }
func (*noopController) UpdateStateID(_ string)                         {}
func (*noopController) StatusString() string                           { return "online" }

type noopReporter struct{}

func (*noopReporter) Update(_ state.Status, _ string) {}
func (*noopReporter) Unregister()                     {}
