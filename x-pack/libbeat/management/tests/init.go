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

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/cmd"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/libbeat/feature"
	lbmanagement "github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/beats/v7/x-pack/libbeat/management"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
	conf "github.com/elastic/elastic-agent-libs/config"
)

var defaultFleetName = "x-pack-fleet"

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

// ResetFleetManager re-registers the global fleet handler, if needed, and replace it with the test one.
func ResetFleetManager(handler MockV2Handler) error {
	managers, err := feature.GlobalRegistry().LookupAll(lbmanagement.Namespace)
	if err != nil {
		return fmt.Errorf("error finding management plugin: %w", err)
	}
	if managers != nil && managers[0].Name() == defaultFleetName {
		_ = feature.GlobalRegistry().Unregister(lbmanagement.Namespace, defaultFleetName)
	}
	lbmanagement.Register("fleet-test", fleetClientFactory(handler), feature.Beta)
	return nil
}

func fleetClientFactory(srv MockV2Handler) lbmanagement.PluginFunc {
	return func(config *conf.C) lbmanagement.FactoryFunc {
		c := management.DefaultConfig()
		if config.Enabled() {
			if err := config.Unpack(&c); err != nil {
				return nil
			}
			return func(_ *conf.C, registry *reload.Registry, beatUUID uuid.UUID) (lbmanagement.Manager, error) {
				return management.NewV2AgentManagerWithClient(c, registry, srv.Client, management.WithStopOnEmptyUnits)
			}
		}
		return nil
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

	server := NewMockServer(t, runtime, config, outPath)
	t.Logf("Resetting fleet manager...")
	err = ResetFleetManager(server)
	require.NoError(t, err)

	return outPath, server
}
