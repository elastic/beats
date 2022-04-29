// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/logp"
	conf "github.com/elastic/elastic-agent-libs/config"
)

func TestBuilder(t *testing.T) {
	provider := "myprovider"
	providerFactory := func(_ *logp.Logger, _ *Registry, _ *conf.C) (Provider, error) {
		return nil, nil
	}

	fnFactory1 := func(_ Provider, _ *conf.C) (Function, error) { return nil, nil }
	fnFactory2 := func(_ Provider, _ *conf.C) (Function, error) { return nil, nil }

	b := MustCreate(
		provider,
		providerFactory,
		feature.MakeDetails("myprovider", "myprovider", feature.Experimental),
	).MustAddFunction(
		"f1",
		fnFactory1,
		feature.MakeDetails("fn1 description", "fn1", feature.Experimental),
	).MustAddFunction("f2", fnFactory2, feature.MakeDetails(
		"fn1 description",
		"fn1",
		feature.Experimental,
	)).Bundle()

	assert.Equal(t, 3, len(b.Features()))
	features := b.Features()

	assert.Equal(t, "myprovider", features[0].Name())
	assert.Equal(t, "functionbeat.provider", features[0].Namespace())

	assert.Equal(t, "f1", features[1].Name())
	assert.Equal(t, "functionbeat.provider.myprovider.functions", features[1].Namespace())

	assert.Equal(t, "f2", features[2].Name())
	assert.Equal(t, "functionbeat.provider.myprovider.functions", features[2].Namespace())
}
