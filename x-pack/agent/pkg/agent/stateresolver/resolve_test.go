// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package stateresolver

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/configrequest"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/transpiler"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/release"
)

func TestResolver(t *testing.T) {
	fb1 := fb("1")
	fb2 := fb("2")
	mb1 := mb("2")
	tn := time.Now()
	tn2 := time.Now().Add(time.Minute * 5)

	testcases := map[string]struct {
		submit cfgReq
		cur    state
		should state
		steps  []configrequest.Step
	}{
		"from no programs to running program": {
			submit: &cfg{
				id:        "config-1",
				createdAt: tn,
				programs: []program.Program{
					fb1, mb1,
				},
			},
			cur: state{}, // empty state
			should: state{
				ID:           "config-1",
				LastModified: tn,
				Active: map[string]active{
					"filebeat": active{
						LastChange:   startState,
						LastModified: tn,
						Identifier:   "filebeat",
						Program:      fb1,
					},
					"metricbeat": active{
						LastChange:   startState,
						LastModified: tn,
						Identifier:   "metricbeat",
						Program:      mb1,
					},
				},
			},
			steps: []configrequest.Step{
				configrequest.Step{
					ID:      configrequest.StepRun,
					Process: fb1.Cmd(),
					Version: release.Version(),
					Meta:    withMeta(fb1),
				},
				configrequest.Step{
					ID:      configrequest.StepRun,
					Process: mb1.Cmd(),
					Version: release.Version(),
					Meta:    withMeta(mb1),
				},
			},
		},
		"adding a program to an already running system": {
			submit: &cfg{
				id:        "config-2",
				createdAt: tn2,
				programs: []program.Program{
					fb1, mb1,
				},
			},
			cur: state{
				ID:           "config-1",
				LastModified: tn,
				Active: map[string]active{
					"filebeat": active{
						LastChange:   startState,
						LastModified: tn,
						Identifier:   "filebeat",
						Program:      fb1,
					},
				},
			},
			should: state{
				ID:           "config-2",
				LastModified: tn2,
				Active: map[string]active{
					"filebeat": active{
						LastChange:   unchangedState,
						LastModified: tn,
						Identifier:   "filebeat",
						Program:      fb1,
					},
					"metricbeat": active{
						LastChange:   startState,
						LastModified: tn2,
						Identifier:   "metricbeat",
						Program:      mb1,
					},
				},
			},
			steps: []configrequest.Step{
				configrequest.Step{
					ID:      configrequest.StepRun,
					Process: mb1.Cmd(),
					Version: release.Version(),
					Meta:    withMeta(mb1),
				},
			},
		},
		"updating an already running program": {
			submit: &cfg{
				id:        "config-2",
				createdAt: tn2,
				programs: []program.Program{
					fb2, mb1,
				},
			},
			cur: state{
				ID:           "config-1",
				LastModified: tn,
				Active: map[string]active{
					"filebeat": active{
						LastChange:   startState,
						LastModified: tn,
						Identifier:   "filebeat",
						Program:      fb1,
					},
				},
			},
			should: state{
				ID:           "config-2",
				LastModified: tn2,
				Active: map[string]active{
					"filebeat": active{
						LastChange:   updateState,
						LastModified: tn2,
						Identifier:   "filebeat",
						Program:      fb2,
					},
					"metricbeat": active{
						LastChange:   startState,
						LastModified: tn2,
						Identifier:   "metricbeat",
						Program:      mb1,
					},
				},
			},
			steps: []configrequest.Step{
				configrequest.Step{
					ID:      configrequest.StepRun,
					Process: fb2.Cmd(),
					Version: release.Version(),
					Meta:    withMeta(fb2),
				},
				configrequest.Step{
					ID:      configrequest.StepRun,
					Process: mb1.Cmd(),
					Version: release.Version(),
					Meta:    withMeta(mb1),
				},
			},
		},
		"remove a running program and start a new one": {
			submit: &cfg{
				id:        "config-2",
				createdAt: tn2,
				programs: []program.Program{
					mb1,
				},
			},
			cur: state{
				ID:           "config-1",
				LastModified: tn,
				Active: map[string]active{
					"filebeat": active{
						LastChange:   startState,
						LastModified: tn,
						Identifier:   "filebeat",
						Program:      fb1,
					},
				},
			},
			should: state{
				ID:           "config-2",
				LastModified: tn2,
				Active: map[string]active{
					"metricbeat": active{
						LastChange:   startState,
						LastModified: tn2,
						Identifier:   "metricbeat",
						Program:      mb1,
					},
				},
			},
			steps: []configrequest.Step{
				configrequest.Step{
					ID:      configrequest.StepRemove,
					Process: fb1.Cmd(),
					Version: release.Version(),
				},
				configrequest.Step{
					ID:      configrequest.StepRun,
					Process: mb1.Cmd(),
					Version: release.Version(),
					Meta:    withMeta(mb1),
				},
			},
		},
		"stops all runnings programs": {
			submit: &cfg{
				id:        "config-2",
				createdAt: tn2,
				programs:  []program.Program{},
			},
			cur: state{
				ID:           "config-1",
				LastModified: tn,
				Active: map[string]active{
					"filebeat": active{
						LastChange:   startState,
						LastModified: tn,
						Identifier:   "filebeat",
						Program:      fb1,
					},
					"metricbeat": active{
						LastChange:   startState,
						LastModified: tn,
						Identifier:   "metricbeat",
						Program:      mb1,
					},
				},
			},
			should: state{
				ID:           "config-2",
				LastModified: tn2,
				Active:       map[string]active{},
			},
			steps: []configrequest.Step{
				configrequest.Step{
					ID:      configrequest.StepRemove,
					Process: fb1.Cmd(),
					Version: release.Version(),
				},
				configrequest.Step{
					ID:      configrequest.StepRemove,
					Process: mb1.Cmd(),
					Version: release.Version(),
				},
			},
		},
		"no changes detected": {
			submit: &cfg{
				id:        "config-1",
				createdAt: tn,
				programs: []program.Program{
					fb1, mb1,
				},
			},
			cur: state{
				ID:           "config-1",
				LastModified: tn,
				Active: map[string]active{
					"filebeat": active{
						LastChange:   startState,
						LastModified: tn,
						Identifier:   "filebeat",
						Program:      fb1,
					},
					"metricbeat": active{
						LastChange:   startState,
						LastModified: tn,
						Identifier:   "metricbeat",
						Program:      mb1,
					},
				},
			},
			should: state{
				ID:           "config-1",
				LastModified: tn,
				Active: map[string]active{
					"filebeat": active{
						LastChange:   unchangedState,
						LastModified: tn,
						Identifier:   "filebeat",
						Program:      fb1,
					},
					"metricbeat": active{
						LastChange:   unchangedState,
						LastModified: tn,
						Identifier:   "metricbeat",
						Program:      mb1,
					},
				},
			},
			steps: []configrequest.Step{},
		},
	}

	for name, test := range testcases {
		t.Run(name, func(t *testing.T) {
			should, steps := converge(test.cur, test.submit)

			require.Equal(t, test.should.ID, should.ID)
			require.Equal(t, test.should.LastModified, should.LastModified)

			require.Equal(t, len(test.steps), len(steps), "steps count don't match")
			require.Equal(t, len(test.should.Active), len(should.Active), "active count don't match")

			for id, a := range test.should.Active {
				compare := should.Active[id]
				require.Equal(t, a.LastModified, compare.LastModified)
				require.Equal(t, a.Identifier, compare.Identifier)
				require.Equal(t, a.LastChange, compare.LastChange)
				require.Equal(t, a.Program.Checksum(), compare.Program.Checksum())
			}

			if diff := cmp.Diff(test.steps, steps); diff != "" {
				t.Errorf("converge() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

type cfg struct {
	id        string
	createdAt time.Time
	programs  []program.Program
}

func (c *cfg) ID() string {
	return c.id
}

func (c *cfg) Programs() []program.Program {
	return c.programs
}

func (c *cfg) CreatedAt() time.Time {
	return c.createdAt
}

func p(identifier, checksum string) program.Program {
	s, ok := program.FindSpecByName(identifier)
	if !ok {
		panic("can't find spec with identifier " + identifier)
	}
	return program.Program{
		Spec: s,
		Config: transpiler.MustNewAST(map[string]interface{}{
			s.Name: map[string]interface{}{
				"checksum": checksum, // make sure checksum is different between configuration change.
			},
		}),
	}
}

func fb(checksum string) program.Program {
	return p("Filebeat", checksum)
}

func mb(checksum string) program.Program {
	return p("Metricbeat", checksum)
}

func withMeta(prog program.Program) map[string]interface{} {
	return map[string]interface{}{
		configrequest.MetaConfigKey: prog.Configuration(),
	}
}
