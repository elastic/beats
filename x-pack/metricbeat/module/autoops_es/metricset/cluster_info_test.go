// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration

package metricset

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
	"github.com/elastic/beats/v7/pkg/logp"
	"github.com/elastic/beats/v7/pkg/logp/logptest"
	libversion "github.com/elastic/beats/v7/pkg/version"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"
)

const unsupportedEsVersion = "7.16.3"

// countFatalErrorHandlers returns the number of goroutines currently running handleFatalErrors.
func countFatalErrorHandlers() int {
	buf := make([]byte, 1<<20)

	for {
		n := runtime.Stack(buf, true)

		if n < len(buf) {
			return strings.Count(string(buf[:n]), "metricset.handleFatalErrors(")
		}

		buf = make([]byte, 2*len(buf))
	}
}

// fatalError is what a fatalErrorHandler received.
type fatalError struct {
	err  error
	code int
}

// captureFatalErrors replaces the handler that stops the agent with one that records what it was
// asked to handle, and resets the version check state when the test ends.
func captureFatalErrors(t *testing.T) <-chan fatalError {
	t.Helper()

	captured := make(chan fatalError, 100)
	original := fatalErrorHandler

	fatalErrorHandler = func(_ *logp.Logger, errChan chan error, errorCode int) {
		captured <- fatalError{err: <-errChan, code: errorCode}
	}

	t.Cleanup(func() {
		fatalErrorHandler = original
		isVersionChecked.Store(false)
	})

	return captured
}

// requireFatalError waits for the handler that stops the agent to be called.
func requireFatalError(t *testing.T, captured <-chan fatalError) fatalError {
	t.Helper()

	select {
	case fatal := <-captured:
		return fatal
	case <-time.After(5 * time.Second):
		t.Fatal("the agent was not stopped")
		return fatalError{}
	}
}

// elasticsearchMetricSet returns the elasticsearch.MetricSet behind the test metricset.
func elasticsearchMetricSet(t *testing.T, data FetcherData[testObjectType]) *elasticsearch.MetricSet {
	t.Helper()

	autoOpsMetricSet, ok := data.MetricSet.(*AutoOpsMetricSet[testObjectType])
	require.True(t, ok, "unexpected metricset type %T", data.MetricSet)

	return autoOpsMetricSet.MetricSet
}

// setupUnsupportedVersionServer ignores the cluster info from the test files and reports a version
// older than MinimumEsVersion.
func setupUnsupportedVersionServer(t *testing.T, _ []byte, data []byte, _ string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.RequestURI {
		case "/":
			_, _ = w.Write([]byte(`{"cluster_name":"my-cluster","cluster_uuid":"GZbSUUMQQI-A7UcGS6vCMa","version":{"number":"` + unsupportedEsVersion + `"}}`))
		case PATH:
			_, _ = w.Write(data)
		default:
			t.Errorf("Unrecognized request %v", r.RequestURI)
		}
	}))
}

// GetInfo runs on every Fetch of every metricset, so anything it leaves behind grows without bound.
// It used to start a goroutine on each call that waited for a version mismatch that never came.
func TestGetInfoDoesNotLeakGoroutines(t *testing.T) {
	const calls = 100

	RunTestsForMetricSetWithGlobFiles(t, "./_meta/test/success.*.json", setupSuccessfulServer, useTestMetricSet, func(t *testing.T, data FetcherData[testObjectType]) {
		m := elasticsearchMetricSet(t, data)
		before := countFatalErrorHandlers()

		for range calls {
			info, err := GetInfo(m)

			require.NoError(t, err)
			require.Equal(t, data.ClusterInfo.ClusterID, info.ClusterID)
		}

		require.Equal(t, before, countFatalErrorHandlers(), "GetInfo must not leave goroutines behind when the version is supported")
		require.False(t, isVersionChecked.Load())
	})
}

// The same through Fetch, which is what the scheduler calls on every period.
func TestFetchDoesNotLeakGoroutines(t *testing.T) {
	const calls = 100

	RunTestsForMetricSetWithGlobFiles(t, "./_meta/test/success.*.json", setupSuccessfulServer, useTestMetricSet, func(t *testing.T, data FetcherData[testObjectType]) {
		before := countFatalErrorHandlers()

		for range calls {
			require.NoError(t, data.MetricSet.Fetch(data.Reporter))
		}

		require.Len(t, data.Reporter.GetEvents(), 2*calls)
		require.Equal(t, before, countFatalErrorHandlers(), "Fetch must not leave goroutines behind when the version is supported")
	})
}

func TestCheckEsVersionSupported(t *testing.T) {
	captured := captureFatalErrors(t)
	logger := logptest.NewTestingLogger(t, "")

	for _, v := range []string{MinimumEsVersion, "7.17.29", "8.19.0", "9.5.5", "10.0.0"} {
		require.NoError(t, checkEsVersion(logger, libversion.MustNew(v)), "version %s is supported", v)
	}

	require.Empty(t, captured, "a supported version must not stop the agent")
	require.False(t, isVersionChecked.Load(), "a supported version must not end the checks")
}

func TestCheckEsVersionUnsupported(t *testing.T) {
	captured := captureFatalErrors(t)
	logger := logptest.NewTestingLogger(t, "")

	err := checkEsVersion(logger, libversion.MustNew(unsupportedEsVersion))

	var mismatch *utils.VersionMismatchError
	require.ErrorAs(t, err, &mismatch)
	require.Equal(t, MinimumEsVersion, mismatch.ExpectedVersion)
	require.Equal(t, unsupportedEsVersion, mismatch.ActualVersion)
	require.True(t, isVersionChecked.Load())

	fatal := requireFatalError(t, captured)
	require.Same(t, mismatch, fatal.err)
	require.Equal(t, CLUSTER_INFO_INITIAL_ERROR, fatal.code)

	// a second check still fails, but the agent is already stopping
	require.Error(t, checkEsVersion(logger, libversion.MustNew(unsupportedEsVersion)))
	require.Empty(t, captured, "the agent must be stopped only once")
}

// Metricsets are fetched concurrently, and each of them checks the version.
func TestCheckEsVersionUnsupportedConcurrently(t *testing.T) {
	const metricsets = 50

	captured := captureFatalErrors(t)
	logger := logptest.NewTestingLogger(t, "")

	var wg sync.WaitGroup

	for range metricsets {
		wg.Go(func() {
			if err := checkEsVersion(logger, libversion.MustNew(unsupportedEsVersion)); err == nil {
				t.Error("the version must not be supported")
			}
		})
	}

	wg.Wait()

	requireFatalError(t, captured)
	require.Empty(t, captured, "the agent must be stopped only once")
}

func TestGetInfoStopsTheAgentOnUnsupportedVersion(t *testing.T) {
	RunTestsForMetricSetWithGlobFiles(t, "./_meta/test/success.9.2.0.json", setupUnsupportedVersionServer, useTestMetricSet, func(t *testing.T, data FetcherData[testObjectType]) {
		captured := captureFatalErrors(t)
		m := elasticsearchMetricSet(t, data)

		info, err := GetInfo(m)

		var mismatch *utils.VersionMismatchError
		require.ErrorAs(t, err, &mismatch)
		require.Nil(t, info)
		require.Equal(t, unsupportedEsVersion, mismatch.ActualVersion)

		fatal := requireFatalError(t, captured)
		require.Same(t, mismatch, fatal.err)
		require.Equal(t, CLUSTER_INFO_INITIAL_ERROR, fatal.code)

		// while the agent is stopping, the version is not checked again
		info, err = GetInfo(m)

		require.NoError(t, err)
		require.Equal(t, unsupportedEsVersion, info.Version.Number.String())
		require.Empty(t, captured, "the agent must be stopped only once")
	})
}
