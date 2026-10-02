// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
// This file was contributed to by generative AI

package fbreceiver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest/observer"

	"github.com/elastic/beats/v7/filebeat/input/file"
	commonfile "github.com/elastic/beats/v7/libbeat/common/file"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/backend/memlog"
	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const registryStatePrefix = "filebeat::logs::"

// TestLogInputReceiversShareRegistry runs two receivers with log inputs over
// one path.data, as Elastic Agent does for the streams of one component, with
// a registry left behind by a previous run. After both stop, the persisted
// offset of each file must be the end of that file: neither receiver may have
// overwritten the other's progress with the stale copy it loaded at startup.
func TestLogInputReceiversShareRegistry(t *testing.T) {
	const lines = 300
	dataDir := t.TempDir()
	logDir := t.TempDir()
	files := []string{filepath.Join(logDir, "a.log"), filepath.Join(logDir, "b.log")}
	for i, f := range files {
		writeFile(t, f, strings.Repeat(fmt.Sprintf("receiver %d line\n", i+1), lines))
	}
	seedRegistry(t, dataDir, files)

	factory := NewFactoryWithSettings(Settings{Home: t.TempDir()})
	receivers := make([]oteltest.ReceiverConfig, len(files))
	for i, f := range files {
		receivers[i] = oteltest.ReceiverConfig{
			Name:    fmt.Sprintf("r%d", i+1),
			Beat:    "filebeat",
			Factory: factory,
			Config: &Config{
				Beatconfig: map[string]any{
					"queue.mem.flush.timeout": "0s",
					"filebeat": map[string]any{
						"inputs": []map[string]any{
							{
								"type":                 "log",
								"enabled":              true,
								"allow_deprecated_use": true,
								"paths":                []string{f},
								"scan_frequency":       "1s",
							},
						},
					},
					"logging":                 map[string]any{"level": "info"},
					"path.home":               t.TempDir(),
					"path.data":               dataDir,
					"http.enabled":            false,
					"management.otel.enabled": true,
				},
			},
		}
	}

	oteltest.CheckReceivers(oteltest.CheckReceiversParams{
		T:         t,
		Receivers: receivers,
		AssertFunc: func(c *assert.CollectT, logs map[string][]mapstr.M, _ *observer.ObservedLogs) {
			for _, rc := range receivers {
				assert.Len(c, logs[rc.Name], lines, "receiver %s must ingest every line of its file", rc.Name)
			}
		},
	})

	offsets := registryOffsets(t, dataDir)
	for _, f := range files {
		info, err := os.Stat(f)
		require.NoError(t, err, "log file must exist")
		assert.Equal(t, info.Size(), offsets[f], "registry must resume %s at its end after both receivers stopped", filepath.Base(f))
	}
}

func openTestRegistry(t *testing.T, dataDir string) (*statestore.Registry, *statestore.Store) {
	t.Helper()
	logger := logptest.NewTestingLogger(t, "")
	backend, err := memlog.New(logger, memlog.Settings{Root: filepath.Join(dataDir, "registry")})
	require.NoError(t, err, "memlog registry must open")
	registry := statestore.NewRegistry(backend)
	store, err := registry.Get("filebeat")
	require.NoError(t, err, "filebeat store must open")
	return registry, store
}

func seedRegistry(t *testing.T, dataDir string, files []string) {
	t.Helper()
	registry, store := openTestRegistry(t, dataDir)
	identifier, err := file.NewStateIdentifier(nil, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err, "native file identifier must be created")
	for _, f := range files {
		info, err := os.Stat(f)
		require.NoError(t, err, "log file must exist")
		st := file.State{Source: f, Finished: true, TTL: -1, Timestamp: time.Now(), FileStateOS: commonfile.GetOSState(info)}
		st.Id, st.IdentifierName = identifier.GenerateID(st)
		require.NoError(t, store.Set(registryStatePrefix+st.Id, st), "seed state for %s must be written", f)
	}
	require.NoError(t, store.Close(), "seed store must close")
	require.NoError(t, registry.Close(), "seed registry must close")
}

func registryOffsets(t *testing.T, dataDir string) map[string]int64 {
	t.Helper()
	registry, store := openTestRegistry(t, dataDir)
	defer registry.Close()
	defer store.Close()
	offsets := map[string]int64{}
	err := store.Each(func(key string, dec statestore.ValueDecoder) (bool, error) {
		if !strings.HasPrefix(key, registryStatePrefix) {
			return true, nil
		}
		var st file.State
		if err := dec.Decode(&st); err != nil {
			return false, err
		}
		offsets[st.Source] = st.Offset
		return true, nil
	})
	require.NoError(t, err, "registry states must be readable")
	return offsets
}
