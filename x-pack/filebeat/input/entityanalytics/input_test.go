// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package entityanalytics

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/unison"
)

type testProvider struct {
	createFn func(c *config.C) (v2.Input, error)
}

func (p *testProvider) Init(grp unison.Group) error {
	return nil
}

func (p *testProvider) Create(c *config.C) (v2.Input, error) {
	if p.createFn != nil {
		return p.createFn(c)
	}

	return nil, nil
}

type testInput struct {
	name   string
	testFn func(testCtx v2.TestContext) error
	runFn  func(runCtx v2.Context, connector beat.PipelineConnector) error
}

func (n *testInput) Name() string {
	return n.name
}

func (n *testInput) Test(testCtx v2.TestContext) error {
	if n.testFn != nil {
		return n.testFn(testCtx)
	}

	return nil
}

func (n *testInput) Run(runCtx v2.Context, connector beat.PipelineConnector) error {
	if n.runFn != nil {
		return n.runFn(runCtx, connector)
	}

	return nil
}

func newTestProvider(input *testProvider) provider.FactoryFunc {
	return func(logger *logp.Logger) (provider.Provider, error) {
		return input, nil
	}
}

func newTestErrProvider() provider.FactoryFunc {
	return func(logger *logp.Logger) (provider.Provider, error) {
		return nil, errors.New("test error")
	}
}

func TestInputManager_Create(t *testing.T) {
	testInputName := "test-input"
	err := provider.Register("test", newTestProvider(&testProvider{
		createFn: func(c *config.C) (v2.Input, error) {
			return &testInput{
				name: testInputName,
			}, nil
		},
	}))
	require.NoError(t, err)

	err = provider.Register("test-err", newTestErrProvider())
	require.NoError(t, err)

	t.Run("create-ok", func(t *testing.T) {
		t.Parallel()

		rawConf := conf{
			Provider: "test",
		}
		c, err := config.NewConfigFrom(&rawConf)
		require.NoError(t, err)

		plugin := Plugin(logp.L())
		inp, err := plugin.Manager.Create(c)
		require.NoError(t, err)
		require.Equal(t, testInputName, inp.Name())
	})

	t.Run("create-err-config", func(t *testing.T) {
		t.Parallel()

		rawConf := conf{
			Provider: "",
		}
		c, err := config.NewConfigFrom(&rawConf)
		require.NoError(t, err)

		plugin := Plugin(logp.L())
		_, err = plugin.Manager.Create(c)

		require.ErrorContains(t, err, "string value is not set accessing 'provider'")
	})

	t.Run("create-err-provider-unknown", func(t *testing.T) {
		t.Parallel()

		rawConf := conf{
			Provider: "foobar",
		}
		c, err := config.NewConfigFrom(&rawConf)
		require.NoError(t, err)

		plugin := Plugin(logp.L())
		_, err = plugin.Manager.Create(c)

		require.ErrorContains(t, err, ErrProviderUnknown.Error())
	})

	t.Run("create-err-provider-create", func(t *testing.T) {
		t.Parallel()

		rawConf := conf{
			Provider: "test-err",
		}
		c, err := config.NewConfigFrom(&rawConf)
		require.NoError(t, err)

		plugin := Plugin(logp.L())
		_, err = plugin.Manager.Create(c)

		require.ErrorContains(t, err, "test error")
	})
}
