// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package oracle_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/metricbeat/mb"
	_ "github.com/elastic/beats/v7/x-pack/metricbeat/module/oracle/tablespace"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/paths"
)

func TestHostParserRedactsInvalidHosts(t *testing.T) {
	tests := map[string]string{
		"invalid port":       "oracle://scott:TigerSecret@dbhost:15x21/svc",
		"invalid escape":     "oracle://scott:TigerSecret%ZZ@127.0.0.1:1521/ORCL",
		"unterminated quote": `user="scott" password="TigerSecret" connectString="dbhost:1521/svc`,
	}

	for name, host := range tests {
		t.Run(name, func(t *testing.T) {
			c, err := conf.NewConfigFrom(map[string]any{
				"module":     "oracle",
				"metricsets": []string{"tablespace"},
				"hosts":      []string{host},
			})
			require.NoError(t, err, "building the module config should not fail")

			_, _, err = mb.NewModule(c, mb.Registry, beat.Info{
				Paths:  &paths.Path{Data: t.TempDir()},
				Logger: logptest.NewTestingLogger(t, ""),
			})
			require.Error(t, err, "host %q should fail to parse", host)
			assert.Contains(t, err.Error(), "error trying to parse connection string", "error should come from the host parser")
			assert.NotContains(t, err.Error(), "TigerSecret", "error must not leak the password from the host")
		})
	}
}
