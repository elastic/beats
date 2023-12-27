// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/cmd"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	lbmanagement "github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/beats/v7/x-pack/libbeat/management"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
	conf "github.com/elastic/elastic-agent-libs/config"
)

// InitBeatsForTest tinkers with a bunch of global variables so beats will start up properly in a test environment
func InitBeatsForTest(t *testing.T, beatRoot *cmd.BeatsRootCmd) {
	// this is a tad hacky, but the go test environment will attempt to insert a bunch of CLI args into the executable,
	// which beats's CLI library will then choke on
	os.Args = os.Args[:1]

	// Set CLI flags needed to run the tests
	t.Logf("Setting flags...")
	err := beatRoot.PersistentFlags().Set("e", "true")
	require.NoError(t, err)
	err = beatRoot.PersistentFlags().Set("E", "management.enabled=true")
	require.NoError(t, err)
	err = beatRoot.PersistentFlags().Set("d", "centralmgmt.V2-manager")
	require.NoError(t, err)
}

func fleetClientFactory(srv MockV2Handler) lbmanagement.ManagerFactory {
	return func(cfg *conf.C, registry *reload.Registry) (lbmanagement.Manager, error) {
		c := management.DefaultConfig()
		if err := cfg.Unpack(&c); err != nil {
			return nil, err
		}
		return management.NewV2AgentManagerWithClient(c, registry, srv.Client, management.WithStopOnEmptyUnits)
	}
}

// SetupTestEnv is a helper to initialize the common files and handlers for metricbeat.
// This returns a string to the tmpdir location
func SetupTestEnv(t *testing.T, config *proto.UnitExpectedConfig, runtime time.Duration) (string, MockV2Handler) {
	tmpdir := os.TempDir()
	filename := fmt.Sprintf("test-%d", time.Now().Unix())
	outPath := filepath.Join(tmpdir, filename)
	t.Logf("writing output to file %s", outPath)
	err := os.Mkdir(outPath, 0775)
	require.NoError(t, err)

	start := time.Now()
	server := NewMockServer(t, func(_ string) bool { return time.Since(start) > runtime }, config, outPath)
	t.Logf("Resetting fleet manager...")
	lbmanagement.SetManagerFactory(fleetClientFactory(server))

	return outPath, server
}
