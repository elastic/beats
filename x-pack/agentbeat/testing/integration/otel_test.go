// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License 2.0;
// you may not use this file except in compliance with the Elastic License 2.0.

//go:build integration

package integration

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent/pkg/control/v2/client"
	aTesting "github.com/elastic/elastic-agent/pkg/testing"
	atesting "github.com/elastic/elastic-agent/pkg/testing"
	integrationtest "github.com/elastic/elastic-agent/pkg/testing"
	"github.com/elastic/elastic-agent/pkg/testing/define"
	"github.com/elastic/elastic-agent/pkg/testing/tools/estools"
	"github.com/elastic/elastic-agent/pkg/testing/tools/testcontext"
	"github.com/elastic/go-elasticsearch/v8"
)

const fileProcessingFilename = `/tmp/testfileprocessing.json`

var fileProcessingConfig = []byte(`receivers:
  filelog:
    include: [ "/var/log/system.log", "/var/log/syslog"  ]
    start_at: beginning

exporters:
  file:
    path: ` + fileProcessingFilename + `
service:
  pipelines:
    logs:
      receivers: [filelog]
      exporters:
        - file`)

var fileInvalidOtelConfig = []byte(`receivers:
  filelog:
    include: [ "/var/log/system.log", "/var/log/syslog"  ]
    start_at: beginning

exporters:
  file:
    path: ` + fileProcessingFilename + `
service:
  pipelines:
    logs:
      receivers: [filelog]
      processors: [nonexistingprocessor]
      exporters:
        - file`)

const apmProcessingContent = `2023-06-19 05:20:50 ERROR This is a test error message
2023-06-20 12:50:00 DEBUG This is a test debug message 2
2023-06-20 12:51:00 DEBUG This is a test debug message 3
2023-06-20 12:52:00 DEBUG This is a test debug message 4`

const apmOtelConfig = `receivers:
  filelog:
    include: [ %s ]
    operators:
      - type: regex_parser
        regex: '^(?P<time>\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}) (?P<sev>[A-Z]*) (?P<msg>.*)$'
        timestamp:
          parse_from: attributes.time
          layout: '%%Y-%%m-%%d %%H:%%M:%%S'
        severity:
          parse_from: attributes.sev

processors:
  resource:
    attributes:
    # APM Server will use service.name for data stream name: logs-apm.app.<service_name>-default
    - key: service.name
      action: insert
      value: elastic-otel-test
    - key: host.test-id
      action: insert
      value: %s

exporters:
  debug:
    verbosity: detailed
    sampling_initial: 10000
    sampling_thereafter: 10000
  otlp/elastic:
      endpoint: "127.0.0.1:8200"
      tls:
        insecure: true

service:
  pipelines:
    logs:
      receivers: [filelog]
      processors: [resource]
      exporters:
        - debug
        - otlp/elastic`

func TestOtelFileProcessing(t *testing.T) {
	define.Require(t, define.Requirements{
		Group: Default,
		Local: true,
		OS: []define.OS{
			// input path missing on windows
			{Type: define.Linux},
			{Type: define.Darwin},
		},
	})

	t.Cleanup(func() {
		_ = os.Remove(fileProcessingFilename)
	})

	// replace default elastic-agent.yml with otel config
	// otel mode should be detected automatically
	tempDir := t.TempDir()
	cfgFilePath := filepath.Join(tempDir, "otel.yml")
	require.NoError(t, os.WriteFile(cfgFilePath, []byte(fileProcessingConfig), 0600))

	fixture, err := define.NewFixtureFromLocalBuild(t, define.Version(), aTesting.WithAdditionalArgs([]string{"--config", cfgFilePath}))
	require.NoError(t, err)

	ctx, cancel := testcontext.WithDeadline(t, context.Background(), time.Now().Add(10*time.Minute))
	defer cancel()
	err = fixture.Prepare(ctx, fakeComponent)
	require.NoError(t, err)

	// remove elastic-agent.yml, otel should be independent
	require.NoError(t, os.Remove(filepath.Join(fixture.WorkDir(), "elastic-agent.yml")))

	var fixtureWg sync.WaitGroup
	fixtureWg.Add(1)
	go func() {
		defer fixtureWg.Done()
		err = fixture.RunOtelWithClient(ctx, false, false)
	}()

	var content []byte
	watchLines := linesTrackMap([]string{
		`"stringValue":"syslog"`,     // syslog is being processed
		`"stringValue":"system.log"`, // system.log is being processed
	})

	validateCommandIsWorking(t, ctx, fixture, tempDir)

	// check `elastic-agent status` returns successfully
	require.Eventuallyf(t, func() bool {
		// This will return errors until it connects to the agent,
		// they're mostly noise because until the agent starts running
		// we will get connection errors. If the test fails
		// the agent logs will be present in the error message
		// which should help to explain why the agent was not
		// healthy.
		err = fixture.IsHealthy(ctx)
		return err == nil
	},
		2*time.Minute, time.Second,
		"Elastic-Agent did not report healthy. Agent status error: \"%v\"",
		err,
	)

	require.Eventually(t,
		func() bool {
			// verify file exists
			content, err = os.ReadFile(fileProcessingFilename)
			if err != nil || len(content) == 0 {
				return false
			}

			for k, alreadyFound := range watchLines {
				if alreadyFound {
					continue
				}
				if bytes.Contains(content, []byte(k)) {
					watchLines[k] = true
				}
			}

			return mapAtLeastOneTrue(watchLines)
		},
		3*time.Minute, 500*time.Millisecond,
		fmt.Sprintf("there should be exported logs by now"))

	testAgentCanRun(ctx, t, fixture)

	cancel()
	fixtureWg.Wait()
	require.True(t, err == nil || err == context.Canceled || err == context.DeadlineExceeded, "Retrieved unexpected error: %s", err.Error())
}

func validateCommandIsWorking(t *testing.T, ctx context.Context, fixture *aTesting.Fixture, tempDir string) {
	cfgFilePath := filepath.Join(tempDir, "otel-valid.yml")
	require.NoError(t, os.WriteFile(cfgFilePath, []byte(fileProcessingConfig), 0600))

	// check `elastic-agent otel validate` command works for otel config
	cmd, err := fixture.PrepareAgentCommand(ctx, []string{"otel", "validate", "--config", cfgFilePath})
	require.NoError(t, err)

	err = cmd.Run()
	require.NoError(t, err)

	// check feature gate works
	out, err := fixture.Exec(ctx, []string{"otel", "validate", "--config", cfgFilePath, "--feature-gates", "foo.bar"})
	require.Error(t, err)
	require.Contains(t, string(out), `no such feature gate "foo.bar"`)

	// check `elastic-agent otel validate` command works for invalid otel config
	cfgFilePath = filepath.Join(tempDir, "otel-invalid.yml")
	require.NoError(t, os.WriteFile(cfgFilePath, []byte(fileInvalidOtelConfig), 0600))

	out, err = fixture.Exec(ctx, []string{"otel", "validate", "--config", cfgFilePath})
	require.Error(t, err)
	require.False(t, len(out) == 0)
	require.Contains(t, string(out), `service::pipelines::logs: references processor "nonexistingprocessor" which is not configured`)
}

var logsIngestionConfigTemplate = `
exporters:
  debug:
    verbosity: basic

  elasticsearch:
    api_key: {{.ESApiKey}}
    endpoint: {{.ESEndpoint}}

processors:
  resource/add-test-id:
    attributes:
    - key: test.id
      action: insert
      value: {{.TestId}}

receivers:
  filelog:
    include:
      - {{.InputFilePath}}
    start_at: beginning

service:
  pipelines:
    logs:
      exporters:
        - debug
        - elasticsearch
      processors:
        - resource/add-test-id
      receivers:
        - filelog
`

func TestOtelLogsIngestion(t *testing.T) {
	info := define.Require(t, define.Requirements{
		Group: Default,
		Local: true,
		OS: []define.OS{
			// input path missing on windows
			{Type: define.Linux},
			{Type: define.Darwin},
		},
		Stack: &define.Stack{},
	})

	// Prepare the OTel config.
	testId := info.Namespace

	tempDir := t.TempDir()
	inputFilePath := filepath.Join(tempDir, "input.log")

	esHost, err := getESHost()
	require.NoError(t, err, "failed to get ES host")
	require.True(t, len(esHost) > 0)

	esClient := info.ESClient
	require.NotNil(t, esClient)
	esApiKey, err := createESApiKey(esClient)
	require.NoError(t, err, "failed to get api key")
	require.True(t, len(esApiKey.Encoded) > 1, "api key is invalid %q", esApiKey)

	logsIngestionConfig := logsIngestionConfigTemplate
	logsIngestionConfig = strings.ReplaceAll(logsIngestionConfig, "{{.ESApiKey}}", esApiKey.Encoded)
	logsIngestionConfig = strings.ReplaceAll(logsIngestionConfig, "{{.ESEndpoint}}", esHost)
	logsIngestionConfig = strings.ReplaceAll(logsIngestionConfig, "{{.InputFilePath}}", inputFilePath)
	logsIngestionConfig = strings.ReplaceAll(logsIngestionConfig, "{{.TestId}}", testId)

	cfgFilePath := filepath.Join(tempDir, "otel.yml")
	require.NoError(t, os.WriteFile(cfgFilePath, []byte(logsIngestionConfig), 0600))

	fixture, err := define.NewFixtureFromLocalBuild(t, define.Version(), aTesting.WithAdditionalArgs([]string{"--config", cfgFilePath}))
	require.NoError(t, err)

	ctx, cancel := testcontext.WithDeadline(t, context.Background(), time.Now().Add(10*time.Minute))
	defer cancel()
	err = fixture.Prepare(ctx, fakeComponent)
	require.NoError(t, err)

	// remove elastic-agent.yml, otel should be independent
	require.NoError(t, os.Remove(filepath.Join(fixture.WorkDir(), "elastic-agent.yml")))

	var fixtureWg sync.WaitGroup
	fixtureWg.Add(1)
	go func() {
		defer fixtureWg.Done()
		err = fixture.RunOtelWithClient(ctx, false, false)
	}()

	validateCommandIsWorking(t, ctx, fixture, tempDir)

	// check `elastic-agent status` returns successfully
	require.Eventuallyf(t, func() bool {
		// This will return errors until it connects to the agent,
		// they're mostly noise because until the agent starts running
		// we will get connection errors. If the test fails
		// the agent logs will be present in the error message
		// which should help to explain why the agent was not
		// healthy.
		err = fixture.IsHealthy(ctx)
		return err == nil
	},
		2*time.Minute, time.Second,
		"Elastic-Agent did not report healthy. Agent status error: \"%v\"",
		err,
	)

	// Write logs to input file.
	logsCount := 10_000
	inputFile, err := os.OpenFile(inputFilePath, os.O_CREATE|os.O_WRONLY, 0600)
	require.NoError(t, err)
	for i := 0; i < logsCount; i++ {
		_, err = fmt.Fprintf(inputFile, "This is a test log message %d\n", i+1)
		require.NoError(t, err)
	}
	inputFile.Close()
	t.Cleanup(func() {
		_ = os.Remove(inputFilePath)
	})

	actualHits := &struct{ Hits int }{}
	require.Eventually(t,
		func() bool {
			findCtx, findCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer findCancel()

			docs, err := estools.GetLogsForIndexWithContext(findCtx, esClient, ".ds-logs-generic-default*", map[string]interface{}{
				"Resource.test.id": testId,
			})
			require.NoError(t, err)

			actualHits.Hits = docs.Hits.Total.Value
			return actualHits.Hits == logsCount
		},
		2*time.Minute, 1*time.Second,
		"Expected %v logs, got %v", logsCount, actualHits)

	cancel()
	fixtureWg.Wait()
	require.True(t, err == nil || err == context.Canceled || err == context.DeadlineExceeded, "Retrieved unexpected error: %s", err.Error())
}

func TestOtelAPMIngestion(t *testing.T) {
	info := define.Require(t, define.Requirements{
		Group: Default,
		Stack: &define.Stack{},
		Local: true,
		OS: []define.OS{
			// apm server not supported on darwin
			{Type: define.Linux},
		},
	})

	const apmVersionMismatch = "The APM integration must be upgraded"
	const apmReadyLog = "all precondition checks are now satisfied"
	logWatcher := aTesting.NewLogWatcher(t,
		apmVersionMismatch, // apm version mismatch
		apmReadyLog,        // apm ready
	)

	// prepare agent
	testId := info.Namespace
	tempDir := t.TempDir()
	cfgFilePath := filepath.Join(tempDir, "otel.yml")
	fileName := "content.log"
	apmConfig := fmt.Sprintf(apmOtelConfig, filepath.Join(tempDir, fileName), testId)
	require.NoError(t, os.WriteFile(cfgFilePath, []byte(apmConfig), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, fileName), []byte{}, 0600))

	fixture, err := define.NewFixtureFromLocalBuild(t, define.Version(), aTesting.WithAdditionalArgs([]string{"--config", cfgFilePath}))
	require.NoError(t, err)

	ctx, cancel := testcontext.WithDeadline(t, context.Background(), time.Now().Add(10*time.Minute))
	defer cancel()
	err = fixture.Prepare(ctx, fakeComponent)
	require.NoError(t, err)

	// prepare input
	agentWorkDir := fixture.WorkDir()

	err = fixture.EnsurePrepared(ctx)
	require.NoError(t, err)

	componentsDir, err := aTesting.FindComponentsDir(agentWorkDir)
	require.NoError(t, err)

	// start apm default config just configure ES output
	esHost, err := getESHost()
	require.NoError(t, err, "failed to get ES host")
	require.True(t, len(esHost) > 0)

	esClient := info.ESClient
	esApiKey, err := createESApiKey(esClient)
	require.NoError(t, err, "failed to get api key")
	require.True(t, len(esApiKey.APIKey) > 1, "api key is invalid %q", esApiKey)

	apmArgs := []string{
		"run",
		"-e",
		"-E", "output.elasticsearch.hosts=['" + esHost + "']",
		"-E", "output.elasticsearch.api_key=" + fmt.Sprintf("%s:%s", esApiKey.Id, esApiKey.APIKey),
		"-E", "apm-server.host=127.0.0.1:8200",
		"-E", "apm-server.ssl.enabled=false",
	}

	apmPath := filepath.Join(componentsDir, "apm-server")
	var apmFixtureWg sync.WaitGroup
	apmFixtureWg.Add(1)
	apmContext, apmCancel := context.WithCancel(ctx)
	defer apmCancel()
	go func() {
		aTesting.RunProcess(t,
			logWatcher,
			apmContext, 0,
			true, true,
			apmPath, apmArgs...)
		apmFixtureWg.Done()
	}()

	// start agent
	var fixtureWg sync.WaitGroup
	fixtureWg.Add(1)
	go func() {
		fixture.RunOtelWithClient(ctx, false, false)
		fixtureWg.Done()
	}()

	// wait for apm to start
	err = logWatcher.WaitForKeys(context.Background(),
		10*time.Minute,
		500*time.Millisecond,
		apmReadyLog,
	)
	require.NoError(t, err, "APM not initialized")

	// wait for otel collector to start
	require.Eventuallyf(t, func() bool {
		// This will return errors until it connects to the agent,
		// they're mostly noise because until the agent starts running
		// we will get connection errors. If the test fails
		// the agent logs will be present in the error message
		// which should help to explain why the agent was not
		// healthy.
		err = fixture.IsHealthy(ctx)
		return err == nil
	},
		2*time.Minute, time.Second,
		"Elastic-Agent did not report healthy. Agent status error: \"%v\"",
		err,
	)

	require.NoError(t, os.WriteFile(filepath.Join(tempDir, fileName), []byte(apmProcessingContent), 0600))

	// check index
	var hits int
	match := map[string]interface{}{
		"labels.host_test-id": testId,
	}

	// apm mismatch or proper docs in ES

	watchLines := linesTrackMap([]string{"This is a test error message",
		"This is a test debug message 2",
		"This is a test debug message 3",
		"This is a test debug message 4"})

	// failed to get APM version mismatch in time
	// processing should be running
	var apmVersionMismatchEncountered bool
	require.Eventually(t,
		func() bool {
			if logWatcher.KeyOccured(apmVersionMismatch) {
				// mark skipped to make it explicit it was not successfully evaluated
				apmVersionMismatchEncountered = true
				return true
			}

			findCtx, findCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer findCancel()
			docs, err := estools.GetLogsForIndexWithContext(findCtx, esClient, "logs-apm*", match)
			if err != nil {
				return false
			}

			hits = len(docs.Hits.Hits)
			if hits <= 0 {
				return false
			}

			for _, hit := range docs.Hits.Hits {
				s, found := hit.Source["message"]
				if !found {
					continue
				}

				for k := range watchLines {
					if strings.Contains(fmt.Sprint(s), k) {
						watchLines[k] = true
					}
				}
			}
			return mapAllTrue(watchLines)
		},
		5*time.Minute, 500*time.Millisecond,
		fmt.Sprintf("there should be apm logs by now: %#v", watchLines))

	if apmVersionMismatchEncountered {
		t.Skip("agent version needs to be equal to stack version")
	}

	// cleanup apm
	cancel()
	apmCancel()
	fixtureWg.Wait()
	apmFixtureWg.Wait()
}

func getESHost() (string, error) {
	fixedESHost := os.Getenv("ELASTICSEARCH_HOST")
	parsedES, err := url.Parse(fixedESHost)
	if err != nil {
		return "", err
	}
	if parsedES.Port() == "" {
		fixedESHost = fmt.Sprintf("%s:443", fixedESHost)
	}
	return fixedESHost, nil
}

func createESApiKey(esClient *elasticsearch.Client) (estools.APIKeyResponse, error) {
	return estools.CreateAPIKey(context.Background(), esClient, estools.APIKeyRequest{Name: "test-api-key", Expiration: "1d"})
}

func linesTrackMap(lines []string) map[string]bool {
	mm := make(map[string]bool)
	for _, l := range lines {
		mm[l] = false
	}
	return mm
}

func mapAllTrue(mm map[string]bool) bool {
	for _, v := range mm {
		if !v {
			return false
		}
	}

	return true
}

func mapAtLeastOneTrue(mm map[string]bool) bool {
	for _, v := range mm {
		if v {
			return true
		}
	}

	return false
}

func testAgentCanRun(ctx context.Context, t *testing.T, fixture *atesting.Fixture) func(*testing.T) {
	tCtx, cancel := testcontext.WithDeadline(t, ctx, time.Now().Add(5*time.Minute))
	defer cancel()

	return func(t *testing.T) {
		// Get path to Elastic Agent executable
		devFixture, err := define.NewFixtureFromLocalBuild(t, define.Version())
		require.NoError(t, err)

		// Prepare the Elastic Agent so the binary is extracted and ready to use.
		err = devFixture.Prepare(tCtx)
		require.NoError(t, err)

		require.Eventually(
			t,
			func() bool {
				return nil == devFixture.Run(tCtx, integrationtest.State{
					Configure:  complexIsolatedUnitsConfig,
					AgentState: integrationtest.NewClientState(client.Healthy),
				})
			},
			5*time.Minute, 500*time.Millisecond,
			"Agent should not return error",
		)
	}
}
