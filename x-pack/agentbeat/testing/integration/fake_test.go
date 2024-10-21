// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License 2.0;
// you may not use this file except in compliance with the Elastic License 2.0.

//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent/pkg/control/v2/client"
	atesting "github.com/elastic/elastic-agent/pkg/testing"
	"github.com/elastic/elastic-agent/pkg/testing/define"
	"github.com/elastic/elastic-agent/pkg/testing/tools/testcontext"
)

var simpleConfig1 = `
outputs:
  default:
    type: fake-output
inputs:
  - id: fake
    type: fake
    state: 1
    message: Configuring
`

var simpleConfig2 = `
outputs:
  default:
    type: fake-output
inputs:
  - id: fake
    type: fake
    state: 2
    message: Healthy
`

var simpleIsolatedUnitsConfig = `
outputs:
  default:
    type: fake-output
inputs:
  - id: fake-isolated-units
    type: fake-isolated-units
    state: 1
    message: Configuring
`

var complexIsolatedUnitsConfig = `
outputs:
  default:
    type: fake-output
inputs:
  - id: fake-isolated-units
    type: fake-isolated-units
    state: 2
    message: Healthy
  - id: fake-isolated-units-1
    type: fake-isolated-units
    state: 2
    message: Healthy
`

func TestFakeComponent(t *testing.T) {
	define.Require(t, define.Requirements{
		Group: Default,
		Local: true,
	})

	f, err := define.NewFixtureFromLocalBuild(t, define.Version())
	require.NoError(t, err)

	ctx, cancel := testcontext.WithDeadline(t, context.Background(), time.Now().Add(10*time.Minute))
	defer cancel()
	err = f.Prepare(ctx, fakeComponent)
	require.NoError(t, err)

	err = f.Run(ctx, atesting.State{
		Configure:  simpleConfig1,
		AgentState: atesting.NewClientState(client.Healthy),
		Components: map[string]atesting.ComponentState{
			"fake-default": {
				State: atesting.NewClientState(client.Healthy),
				Units: map[atesting.ComponentUnitKey]atesting.ComponentUnitState{
					atesting.ComponentUnitKey{UnitType: client.UnitTypeOutput, UnitID: "fake-default"}: {
						State: atesting.NewClientState(client.Healthy),
					},
					atesting.ComponentUnitKey{UnitType: client.UnitTypeInput, UnitID: "fake-default-fake"}: {
						State: atesting.NewClientState(client.Configuring),
					},
				},
			},
		},
	}, atesting.State{
		Configure:  simpleConfig2,
		AgentState: atesting.NewClientState(client.Healthy),
		StrictComponents: map[string]atesting.ComponentState{
			"fake-default": {
				State: atesting.NewClientState(client.Healthy),
				Units: map[atesting.ComponentUnitKey]atesting.ComponentUnitState{
					atesting.ComponentUnitKey{UnitType: client.UnitTypeOutput, UnitID: "fake-default"}: {
						State: atesting.NewClientState(client.Healthy),
					},
					atesting.ComponentUnitKey{UnitType: client.UnitTypeInput, UnitID: "fake-default-fake"}: {
						State: atesting.NewClientState(client.Healthy),
					},
				},
			},
		},
	})
	require.NoError(t, err)
}

func TestFakeIsolatedUnitsComponent(t *testing.T) {
	define.Require(t, define.Requirements{
		Group: Default,
		Local: true,
	})

	f, err := define.NewFixtureFromLocalBuild(t, define.Version())
	require.NoError(t, err)

	ctx, cancel := testcontext.WithDeadline(t, context.Background(), time.Now().Add(10*time.Minute))
	defer cancel()
	err = f.Prepare(ctx, fakeComponent)
	require.NoError(t, err)

	err = f.Run(ctx, atesting.State{
		Configure:  simpleIsolatedUnitsConfig,
		AgentState: atesting.NewClientState(client.Healthy),
		Components: map[string]atesting.ComponentState{
			"fake-isolated-units-default-fake-isolated-units": {
				State: atesting.NewClientState(client.Healthy),
				Units: map[atesting.ComponentUnitKey]atesting.ComponentUnitState{
					atesting.ComponentUnitKey{UnitType: client.UnitTypeOutput, UnitID: "fake-isolated-units-default-fake-isolated-units"}: {
						State: atesting.NewClientState(client.Healthy),
					},
					atesting.ComponentUnitKey{UnitType: client.UnitTypeInput, UnitID: "fake-isolated-units-default-fake-isolated-units-unit"}: {
						State: atesting.NewClientState(client.Configuring),
					},
				},
			},
		},
	}, atesting.State{
		Configure:  complexIsolatedUnitsConfig,
		AgentState: atesting.NewClientState(client.Healthy),
		Components: map[string]atesting.ComponentState{
			"fake-isolated-units-default-fake-isolated-units": {
				State: atesting.NewClientState(client.Healthy),
				Units: map[atesting.ComponentUnitKey]atesting.ComponentUnitState{
					atesting.ComponentUnitKey{UnitType: client.UnitTypeOutput, UnitID: "fake-isolated-units-default-fake-isolated-units"}: {
						State: atesting.NewClientState(client.Healthy),
					},
					atesting.ComponentUnitKey{UnitType: client.UnitTypeInput, UnitID: "fake-isolated-units-default-fake-isolated-units-unit"}: {
						State: atesting.NewClientState(client.Healthy),
					},
				},
			},
			"fake-isolated-units-default-fake-isolated-units-1": {
				State: atesting.NewClientState(client.Healthy),
				Units: map[atesting.ComponentUnitKey]atesting.ComponentUnitState{
					atesting.ComponentUnitKey{UnitType: client.UnitTypeOutput, UnitID: "fake-isolated-units-default-fake-isolated-units-1"}: {
						State: atesting.NewClientState(client.Healthy),
					},
					atesting.ComponentUnitKey{UnitType: client.UnitTypeInput, UnitID: "fake-isolated-units-default-fake-isolated-units-1-unit"}: {
						State: atesting.NewClientState(client.Healthy),
					},
				},
			},
		},
	})
	require.NoError(t, err)
}
