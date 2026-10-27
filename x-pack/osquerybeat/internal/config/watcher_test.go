// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package config

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func TestReloadRejectsInvalidPublishing(t *testing.T) {
	ch := make(chan []InputConfig, 1)
	r := &reloader{ctx: t.Context(), log: logptest.NewTestingLogger(t, "reloader"), ch: ch}
	bad := conf.MustNewConfigFrom(map[string]any{
		"osquery": map[string]any{"elastic_options": map[string]any{"publishing": map[string]any{
			"scheduled": map[string]any{"max_events": -1},
		}}},
	})
	err := r.Reload([]*reload.ConfigWithMeta{{Config: bad, InputUnitID: "osquery-unit"}})
	require.Error(t, err, "a policy with a negative max_events must be rejected")
	assert.Contains(t, err.Error(), "publishing.scheduled.max_events", "the error must name the offending setting")
	// The V2 manager reports only cfgfile.UnitError values found in an error
	// list; any other reload error is discarded without a log or status.
	errs, ok := err.(interface{ Unwrap() []error })
	require.True(t, ok, "the error must be an error list for the manager to inspect it")
	require.Len(t, errs.Unwrap(), 1, "one rejected unit")
	var unitErr cfgfile.UnitError
	require.ErrorAs(t, errs.Unwrap()[0], &unitErr, "the error must identify the rejected unit")
	assert.Equal(t, "osquery-unit", unitErr.UnitID, "the error must name the unit's ID")
	assert.Empty(t, ch, "a rejected policy must not reach the beater")
}

func TestReloadRejectsPolicyThatFailsValidation(t *testing.T) {
	errInvalid := errors.New("invalid ECS mapping")
	ch := make(chan []InputConfig, 1)
	r := &reloader{
		ctx: t.Context(), log: logptest.NewTestingLogger(t, "reloader"), ch: ch,
		validate: func(inputs []InputConfig) error {
			if inputs[0].Datastream.Namespace == "bad" {
				return errInvalid
			}
			return nil
		},
	}
	policy := func(namespace string) []*reload.ConfigWithMeta {
		return []*reload.ConfigWithMeta{{InputUnitID: "osquery-unit", Config: conf.MustNewConfigFrom(map[string]any{
			"data_stream": map[string]any{"namespace": namespace},
		})}}
	}

	err := r.Reload(policy("bad"))
	require.Error(t, err, "a policy that fails validation must be rejected")
	errs, ok := err.(interface{ Unwrap() []error })
	require.True(t, ok, "the error must be an error list for the manager to inspect it")
	require.Len(t, errs.Unwrap(), 1, "one rejected unit")
	var unitErr cfgfile.UnitError
	require.ErrorAs(t, errs.Unwrap()[0], &unitErr, "the error must identify the rejected unit")
	assert.Equal(t, "osquery-unit", unitErr.UnitID, "the error must name the unit's ID")
	assert.ErrorIs(t, err, errInvalid, "the error must carry the validation error")
	assert.Empty(t, ch, "a rejected policy must not reach the beater")

	require.NoError(t, r.Reload(policy("good")), "a valid policy after a rejected one must be accepted")
	require.Len(t, ch, 1, "the valid policy must reach the beater")
	assert.Equal(t, "good", (<-ch)[0].Datastream.Namespace, "the beater must get the valid policy")
}
