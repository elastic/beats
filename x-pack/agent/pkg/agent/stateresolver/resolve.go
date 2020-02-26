// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package stateresolver

import (
	"sort"
	"strings"
	"time"

	"github.com/elastic/beats/x-pack/agent/pkg/agent/configrequest"
	"github.com/elastic/beats/x-pack/agent/pkg/agent/program"
	"github.com/elastic/beats/x-pack/agent/pkg/release"
)

//go:generate stringer -type=stateChange -linecomment=true

const shortID = 8

// stateChange represent a how a process is modified between configuration change.
type stateChange uint8

const (
	startState     stateChange = iota + 1 // START
	updateState                           // UPDATE
	unchangedState                        // UNCHANGED
)

type id string

// state represent the SHOULD state of the system, contains a reference to the actual bundle of
// configuration received by the upstream call and keep track of the last change executed on a program.
//
// The list of change are the following:
// start: first time to see that configuration and decide to start a new process.
// update: need to update the process switch a new configuration.
// unchanged: keep running the process with the actual configuration.
type state struct {
	ID           string
	LastModified time.Time
	Active       map[string]active
}

func (s *state) ShortID() string {
	if len(s.ID) <= shortID {
		return s.ID
	}
	return s.ID[0:shortID]
}

func (s *state) String() string {
	var str strings.Builder
	str.WriteString("ID:" + s.ID + ", LastModified: " + s.LastModified.String())
	str.WriteString("Active Process [\n")
	for _, a := range s.Active {
		str.WriteString(a.String())
	}
	str.WriteString("]")

	return str.String()
}

type active struct {
	LastChange   stateChange
	LastModified time.Time
	Identifier   string
	Program      program.Program
}

func (s *active) String() string {
	return "Identifier: " + s.Identifier +
		", LastChange: " + s.LastChange.String() +
		", LastModified: " + s.LastModified.String() +
		", Checksum: " + s.Program.Checksum()
}

type cfgReq interface {
	ID() string
	CreatedAt() time.Time
	Programs() []program.Program
}

// Converge converges the system, take the current sate and create a new should state and all the steps
// required to go from current state to the new state.
func converge(s state, cfg cfgReq) (state, []configrequest.Step) {
	newState := state{
		ID:           cfg.ID(),
		LastModified: cfg.CreatedAt(),
		Active:       make(map[string]active, len(cfg.Programs())),
	}

	steps := make([]configrequest.Step, 0)

	// Find process that must be stopped.
	activeKeys := getActiveKeys(s.Active)
	for _, id := range activeKeys {
		active := s.Active[id]

		var found bool
		for _, p := range cfg.Programs() {
			// Still need to run the process.
			if id == p.Identifier() {
				found = true
				break
			}
		}

		if !found {
			steps = append(steps, configrequest.Step{
				ID:      configrequest.StepRemove,
				Process: active.Program.Cmd(),
				Version: release.Version(),
			})
		}
	}

	// What need to be started or updated.
	for _, p := range cfg.Programs() {
		a, found := s.Active[p.Identifier()]
		if !found {
			newState.Active[p.Identifier()] = active{
				LastChange:   startState,
				LastModified: cfg.CreatedAt(),
				Identifier:   p.Identifier(),
				Program:      p,
			}

			steps = append(steps, configrequest.Step{
				ID:      configrequest.StepRun,
				Process: p.Cmd(),
				Version: release.Version(),
				Meta: map[string]interface{}{
					configrequest.MetaConfigKey: p.Configuration(),
				},
			})

			// Complete new process, skip to the next process.
			continue
		}

		// Checksum doesn't match and we force an update of the process.
		if a.Program.Checksum() != p.Checksum() {
			newState.Active[p.Identifier()] = active{
				LastChange:   updateState,
				LastModified: cfg.CreatedAt(),
				Identifier:   p.Identifier(),
				Program:      p,
			}
			steps = append(steps, configrequest.Step{
				ID:      configrequest.StepRun,
				Process: p.Cmd(),
				Version: release.Version(),
				Meta: map[string]interface{}{
					configrequest.MetaConfigKey: p.Configuration(),
				},
			})
		} else {
			// Configuration did not change in this loop so we keep
			// the last configuration as is.
			a.LastChange = unchangedState
			newState.Active[p.Identifier()] = a
		}
	}

	// What need to be updated.
	return newState, steps
}

func getActiveKeys(aa map[string]active) []string {
	keys := make([]string, 0, len(aa))
	for k := range aa {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}
