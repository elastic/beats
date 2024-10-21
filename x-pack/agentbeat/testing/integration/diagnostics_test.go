// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License 2.0;
// you may not use this file except in compliance with the Elastic License 2.0.

//go:build integration

package integration

import (
	"archive/zip"
	"context"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent/pkg/control/v2/client"
	integrationtest "github.com/elastic/elastic-agent/pkg/testing"
	"github.com/elastic/elastic-agent/pkg/testing/define"
	"github.com/elastic/elastic-agent/pkg/testing/tools/testcontext"
)

const diagnosticsArchiveGlobPattern = "elastic-agent-diagnostics-*.zip"

var diagnosticsFiles = []string{
	"package.version",
	"agent-info.yaml",
	"allocs.pprof.gz",
	"block.pprof.gz",
	"components-actual.yaml",
	"components-expected.yaml",
	"computed-config.yaml",
	"goroutine.pprof.gz",
	"heap.pprof.gz",
	"local-config.yaml",
	"mutex.pprof.gz",
	"pre-config.yaml",
	"local-config.yaml",
	"state.yaml",
	"threadcreate.pprof.gz",
	"variables.yaml",
	"version.txt",
}

var compDiagnosticsFiles = []string{
	"allocs.pprof.gz",
	"block.pprof.gz",
	"goroutine.pprof.gz",
	"heap.pprof.gz",
	"mutex.pprof.gz",
	"threadcreate.pprof.gz",
}

var componentSetup = map[string]integrationtest.ComponentState{
	"fake-default": {
		State: integrationtest.NewClientState(client.Healthy),
		Units: map[integrationtest.ComponentUnitKey]integrationtest.ComponentUnitState{
			integrationtest.ComponentUnitKey{UnitType: client.UnitTypeOutput, UnitID: "fake-default"}: {
				State: integrationtest.NewClientState(client.Healthy),
			},
			integrationtest.ComponentUnitKey{UnitType: client.UnitTypeInput, UnitID: "fake-default-fake"}: {
				State: integrationtest.NewClientState(client.Healthy),
			},
		},
	},
}

var isolatedUnitsComponentSetup = map[string]integrationtest.ComponentState{
	"fake-isolated-units-default-fake-isolated-units": {
		State: integrationtest.NewClientState(client.Healthy),
		Units: map[integrationtest.ComponentUnitKey]integrationtest.ComponentUnitState{
			integrationtest.ComponentUnitKey{UnitType: client.UnitTypeOutput, UnitID: "fake-isolated-units-default-fake-isolated-units"}: {
				State: integrationtest.NewClientState(client.Healthy),
			},
			integrationtest.ComponentUnitKey{UnitType: client.UnitTypeInput, UnitID: "fake-isolated-units-default-fake-isolated-units-unit"}: {
				State: integrationtest.NewClientState(client.Healthy),
			},
		},
	},
	"fake-isolated-units-default-fake-isolated-units-1": {
		State: integrationtest.NewClientState(client.Healthy),
		Units: map[integrationtest.ComponentUnitKey]integrationtest.ComponentUnitState{
			integrationtest.ComponentUnitKey{UnitType: client.UnitTypeOutput, UnitID: "fake-isolated-units-default-fake-isolated-units-1"}: {
				State: integrationtest.NewClientState(client.Healthy),
			},
			integrationtest.ComponentUnitKey{UnitType: client.UnitTypeInput, UnitID: "fake-isolated-units-default-fake-isolated-units-1-unit"}: {
				State: integrationtest.NewClientState(client.Healthy),
			},
		},
	},
}

type componentAndUnitNames struct {
	name      string
	unitNames []string
}

func TestDiagnosticsOptionalValues(t *testing.T) {
	define.Require(t, define.Requirements{
		Group: Default,
		Local: false,
	})

	fixture, err := define.NewFixtureFromLocalBuild(t, define.Version())
	require.NoError(t, err)

	ctx, cancel := testcontext.WithDeadline(t, context.Background(), time.Now().Add(10*time.Minute))
	defer cancel()
	err = fixture.Prepare(ctx, fakeComponent)
	require.NoError(t, err)

	diagpprof := append(diagnosticsFiles, "cpu.pprof")
	diagCompPprof := append(compDiagnosticsFiles, "cpu.pprof")

	err = fixture.Run(ctx, integrationtest.State{
		Configure:  simpleConfig2,
		AgentState: integrationtest.NewClientState(client.Healthy),
		Components: componentSetup,
		After:      testDiagnosticsFactory(t, componentSetup, diagpprof, diagCompPprof, fixture, []string{"diagnostics", "-p"}),
	})
	require.NoError(t, err)
}

func TestIsolatedUnitsDiagnosticsOptionalValues(t *testing.T) {
	define.Require(t, define.Requirements{
		Group: Default,
		Local: false,
	})

	fixture, err := define.NewFixtureFromLocalBuild(t, define.Version())
	require.NoError(t, err)

	ctx, cancel := testcontext.WithDeadline(t, context.Background(), time.Now().Add(10*time.Minute))
	defer cancel()
	err = fixture.Prepare(ctx, fakeComponent)
	require.NoError(t, err)

	diagpprof := append(diagnosticsFiles, "cpu.pprof")
	diagCompPprof := append(compDiagnosticsFiles, "cpu.pprof")

	err = fixture.Run(ctx, integrationtest.State{
		Configure:  complexIsolatedUnitsConfig,
		AgentState: integrationtest.NewClientState(client.Healthy),
		Components: isolatedUnitsComponentSetup,
		After:      testDiagnosticsFactory(t, isolatedUnitsComponentSetup, diagpprof, diagCompPprof, fixture, []string{"diagnostics", "-p"}),
	})
	require.NoError(t, err)
}

func TestDiagnosticsCommand(t *testing.T) {
	define.Require(t, define.Requirements{
		Group: Default,
		Local: false,
	})

	f, err := define.NewFixtureFromLocalBuild(t, define.Version())
	require.NoError(t, err)

	ctx, cancel := testcontext.WithDeadline(t, context.Background(), time.Now().Add(10*time.Minute))
	defer cancel()
	err = f.Prepare(ctx, fakeComponent)
	require.NoError(t, err)

	err = f.Run(ctx, integrationtest.State{
		Configure:  simpleConfig2,
		AgentState: integrationtest.NewClientState(client.Healthy),
		Components: componentSetup,
		After:      testDiagnosticsFactory(t, componentSetup, diagnosticsFiles, compDiagnosticsFiles, f, []string{"diagnostics", "collect"}),
	})
	assert.NoError(t, err)
}

func TestIsolatedUnitsDiagnosticsCommand(t *testing.T) {
	define.Require(t, define.Requirements{
		Group: Default,
		Local: false,
	})

	f, err := define.NewFixtureFromLocalBuild(t, define.Version())
	require.NoError(t, err)

	ctx, cancel := testcontext.WithDeadline(t, context.Background(), time.Now().Add(10*time.Minute))
	defer cancel()
	err = f.Prepare(ctx, fakeComponent)
	require.NoError(t, err)

	err = f.Run(ctx, integrationtest.State{
		Configure:  complexIsolatedUnitsConfig,
		AgentState: integrationtest.NewClientState(client.Healthy),
		Components: isolatedUnitsComponentSetup,
		After:      testDiagnosticsFactory(t, isolatedUnitsComponentSetup, diagnosticsFiles, compDiagnosticsFiles, f, []string{"diagnostics", "collect"}),
	})
	assert.NoError(t, err)
}

func testDiagnosticsFactory(t *testing.T, compSetup map[string]integrationtest.ComponentState, diagFiles []string, diagCompFiles []string, fix *integrationtest.Fixture, cmd []string) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		diagZip, err := fix.ExecDiagnostics(ctx, cmd...)

		// get the version of the running agent
		avi, err := getRunningAgentVersion(ctx, fix)
		require.NoError(t, err)

		verifyDiagnosticArchive(t, compSetup, diagZip, diagFiles, diagCompFiles, avi)

		return nil
	}
}

func verifyDiagnosticArchive(t *testing.T, compSetup map[string]integrationtest.ComponentState, diagArchive string, diagFiles []string, diagCompFiles []string, avi *client.Version) {
	// check that the archive is not an empty file
	stat, err := os.Stat(diagArchive)
	require.NoErrorf(t, err, "stat file %q failed", diagArchive)
	require.Greaterf(t, stat.Size(), int64(0), "file %s has incorrect size", diagArchive)

	// extract the zip file into a temp folder
	extractionDir := t.TempDir()

	extractZipArchive(t, diagArchive, extractionDir)

	compAndUnitNames := extractComponentAndUnitNames(compSetup)
	expectedDiagArchiveFilePatterns := compileExpectedDiagnosticFilePatterns(avi, diagFiles, diagCompFiles, compAndUnitNames)

	expectedExtractedFiles := map[string]struct{}{}
	for _, filePattern := range expectedDiagArchiveFilePatterns {
		absFilePattern := filepath.Join(extractionDir, filePattern.pattern)
		files, err := filepath.Glob(absFilePattern)
		assert.NoErrorf(t, err, "error globbing with pattern %q", absFilePattern)
		min := 0
		if filePattern.optional {
			min = -1
		}
		assert.Greaterf(t, len(files), min, "glob pattern %q matched no files", absFilePattern)
		for _, f := range files {
			expectedExtractedFiles[f] = struct{}{}
		}
	}

	actualExtractedDiagFiles := map[string]struct{}{}

	err = filepath.WalkDir(extractionDir, func(path string, entry fs.DirEntry, err error) error {
		require.NoErrorf(t, err, "error walking extracted path %q", path)

		// we are not interested in directories
		if !entry.IsDir() {
			actualExtractedDiagFiles[path] = struct{}{}
			info, err := entry.Info()
			require.NoError(t, err, path)
			assert.Greaterf(t, info.Size(), int64(0), "file %q has an invalid size", path)
		}

		return nil
	})
	require.NoErrorf(t, err, "error walking output directory %q", extractionDir)

	assert.ElementsMatch(t, extractKeysFromMap(expectedExtractedFiles), extractKeysFromMap(actualExtractedDiagFiles))
}

func extractComponentAndUnitNames(compSetup map[string]integrationtest.ComponentState) []componentAndUnitNames {
	comps := make([]componentAndUnitNames, 0, len(compSetup))
	for compName, compState := range compSetup {
		unitNames := make([]string, 0, len(compState.Units))
		for unitKey := range compState.Units {
			unitNames = append(unitNames, unitKey.UnitID)
		}
		comps = append(comps, componentAndUnitNames{
			name:      compName,
			unitNames: unitNames,
		})
	}
	return comps
}

func extractZipArchive(t *testing.T, zipFile string, dst string) {
	t.Helper()

	zReader, err := zip.OpenReader(zipFile)
	require.NoErrorf(t, err, "file %q is not a valid zip archive", zipFile)
	defer zReader.Close()

	t.Logf("extracting diagnostic archive in dir %q", dst)
	for _, zf := range zReader.File {
		filePath := filepath.Join(dst, zf.Name)
		t.Logf("unzipping file %q", filePath)
		require.Truef(t, strings.HasPrefix(filePath, filepath.Clean(dst)+string(os.PathSeparator)), "file %q points outside of extraction dir %q", filePath, dst)

		if zf.FileInfo().IsDir() {
			t.Logf("creating directory %q", filePath)
			err := os.MkdirAll(filePath, os.ModePerm)
			assert.NoError(t, err)
			continue
		}

		err = os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
		require.NoErrorf(t, err, "error creating parent folder for file %q", filePath)

		extractSingleFileFromArchive(t, zf, filePath)

	}
}

func extractSingleFileFromArchive(t *testing.T, src *zip.File, dst string) {
	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, src.Mode())
	require.NoErrorf(t, err, "error creating extracted file %q", dst)

	defer dstFile.Close()

	srcFile, err := src.Open()
	require.NoErrorf(t, err, "error opening zipped file %q", src.Name)

	defer srcFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	require.NoErrorf(t, err, "error copying content from zipped file %q to extracted file %q", src.Name, dst)
}

func getRunningAgentVersion(ctx context.Context, f *integrationtest.Fixture) (*client.Version, error) {
	avi, err := f.Client().Version(ctx)
	if err != nil {
		return nil, err
	}

	return &avi, err
}

func compileExpectedDiagnosticFilePatterns(avi *client.Version, diagfiles []string, diagCompFiles []string, comps []componentAndUnitNames) []filePattern {
	files := make([]filePattern, 0, len(diagnosticsFiles)+len(comps)*len(compDiagnosticsFiles))

	for _, file := range diagfiles {
		files = append(files, filePattern{
			pattern:  file,
			optional: false,
		})
	}

	for _, comp := range comps {
		compPath := path.Join("components", comp.name)
		for _, fileName := range diagCompFiles {
			files = append(files,
				filePattern{
					pattern:  path.Join(compPath, fileName),
					optional: false,
				})
		}
	}

	files = append(files, filePattern{
		pattern:  path.Join("logs", "elastic-agent-"+avi.Commit[:6], "elastic-agent-*.ndjson"),
		optional: false,
	})
	// this pattern overlaps with the previous one but filepath.Glob() does not seem to match using '?' wildcard
	// optional: it doesn't have to be there (in some cases the watcher has not written any logs)
	files = append(files, filePattern{
		pattern:  path.Join("logs", "elastic-agent-"+avi.Commit[:6], "elastic-agent-watcher-*.ndjson"),
		optional: true,
	})

	return files
}

func extractKeysFromMap[K comparable, V any](src map[K]V) []K {
	keys := make([]K, 0, len(src))
	for k := range src {
		keys = append(keys, k)
	}
	return keys
}

type filePattern struct {
	pattern  string
	optional bool
}
