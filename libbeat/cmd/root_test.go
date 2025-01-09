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

//go:build !integration

package cmd

import (
	"crypto/tls"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/cmd/instance"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

type MockBeater struct {
	mock.Mock
}

func (m *MockBeater) Run(b *beat.Beat) error {
	args := m.Called(b)
	return args.Error(0)
}

func (m *MockBeater) Stop() {
	m.Called()
}

func genMockCreator(m *MockBeater) beat.Creator {
	return func(b *beat.Beat, c *config.C) (beat.Beater, error) {
		return m, nil
	}
}

func TestGenRootCmdWithSettings_TLSDefaults(t *testing.T) {
	mb := &MockBeater{}
	settings := instance.Settings{}
	_ = GenRootCmdWithSettings(genMockCreator(mb), settings)

	t.Run("Test defaults", func(t *testing.T) {
		b, err := instance.NewBeat("mockbeat", "testidx", "0.9", false, nil)
		require.NoError(t, err)
		cfg, err := cfgfile.Load(filepath.Join("instance", "testdata", "tls.yml"), nil)
		require.NoError(t, err)
		err = cfg.Unpack(&b.Config)
		require.NoError(t, err)
		assert.True(t, b.Config.Output.IsSet())
		sslCfg, err := b.Config.Output.Config().Child("ssl", -1)
		require.NoError(t, err)
		var common tlscommon.Config
		err = sslCfg.Unpack(&common)
		require.NoError(t, err)
		tlsCfg, err := tlscommon.LoadTLSConfig(&common)
		require.NoError(t, err)

		c := tlsCfg.ToConfig()
		assert.Equal(t, uint16(tls.VersionTLS11), c.MinVersion)
		assert.Equal(t, uint16(tls.VersionTLS13), c.MaxVersion)
	})

	t.Run("Set min TLSv1.0", func(t *testing.T) {
		b, err := instance.NewBeat("mockbeat", "testidx", "0.9", false, nil)
		require.NoError(t, err)

		cfg, err := cfgfile.Load(filepath.Join("instance", "testdata", "tls10.yml"), nil)
		require.NoError(t, err)
		err = cfg.Unpack(&b.Config)
		require.NoError(t, err)
		assert.True(t, b.Config.Output.IsSet())
		sslCfg, err := b.Config.Output.Config().Child("ssl", -1)
		require.NoError(t, err)
		var common tlscommon.Config
		err = sslCfg.Unpack(&common)
		require.NoError(t, err)
		tlsCfg, err := tlscommon.LoadTLSConfig(&common)
		require.NoError(t, err)

		c := tlsCfg.ToConfig()
		assert.Equal(t, uint16(tls.VersionTLS10), c.MinVersion)
		assert.Equal(t, uint16(tls.VersionTLS10), c.MaxVersion)
	})
}
