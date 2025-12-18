// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beatprocessor_test

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/elastic/beats/v7/x-pack/libbeat/common/otelbeat/oteltestcol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/otelcol"
)

func BenchmarkBeatProcessor(b *testing.B) {
	configTemplate := `
service:
  pipelines:
    logs:
      receivers:
        - filebeatreceiver
      processors: _OTEL_PROCESSORS_PIPELINE_
      exporters:
        - debug
  telemetry:
    logs:
      sampling:
        enabled: false
    metrics:
      level: none
receivers:
  filebeatreceiver:
    filebeat:
      inputs:
        - type: benchmark
          id: produce-events
          count: _NUM_EVENTS_
          message: "event-from-benchmark-input"
    processors: _BEAT_PROCESSORS_
    queue.mem.flush.timeout: 0s
    path.home: _PATH_HOME_
processors: _OTEL_PROCESSORS_DEFINITION_
exporters:
  debug:
    verbosity: normal
`

	testCases := []struct {
		name         string
		createConfig func(configTemplate string) string
	}{
		{
			name: "NoProcessor",
			createConfig: func(configTemplate string) string {
				config := configTemplate
				config = strings.ReplaceAll(config, "_BEAT_PROCESSORS_", "[]")
				config = strings.ReplaceAll(config, "_OTEL_PROCESSORS_PIPELINE_", "[]")
				config = strings.ReplaceAll(config, "_OTEL_PROCESSORS_DEFINITION_", "{}")
				config = strings.ReplaceAll(config, "_PATH_HOME_", b.TempDir())
				return config
			},
		},
		{
			name: "InReceiver",
			createConfig: func(configTemplate string) string {
				config := configTemplate
				config = strings.ReplaceAll(config, "_BEAT_PROCESSORS_", `
      - add_fields:
          fields:
            custom_field: custom-value
`)
				config = strings.ReplaceAll(config, "_OTEL_PROCESSORS_PIPELINE_", "[]")
				config = strings.ReplaceAll(config, "_OTEL_PROCESSORS_DEFINITION_", "{}")
				config = strings.ReplaceAll(config, "_PATH_HOME_", b.TempDir())
				return config
			},
		},
		{
			name: "InProcessor",
			createConfig: func(configTemplate string) string {
				config := configTemplate
				config = strings.ReplaceAll(config, "_BEAT_PROCESSORS_", "[]")
				config = strings.ReplaceAll(config, "_OTEL_PROCESSORS_PIPELINE_", "[beat]")
				config = strings.ReplaceAll(config, "_OTEL_PROCESSORS_DEFINITION_", `
  beat:
    processors:
      - add_fields:
          fields:
            custom_field: custom-value
`)
				config = strings.ReplaceAll(config, "_PATH_HOME_", b.TempDir())
				return config
			},
		},
	}

	for _, numEvents := range []int{1, 10, 100} {
		for _, tc := range testCases {
			testName := fmt.Sprintf("%d_Events/%s", numEvents, tc.name)
			configTemplateWithNumEvents := strings.ReplaceAll(configTemplate, "_NUM_EVENTS_", fmt.Sprintf("%d", numEvents))
			b.Run(testName, func(b *testing.B) {
				config := tc.createConfig(configTemplateWithNumEvents)

				for b.Loop() {
					b.StopTimer()
					col := oteltestcol.New(b, config)
					b.StartTimer()

					var wg sync.WaitGroup
					wg.Add(1)
					go func() {
						defer wg.Done()
						ctx, cancel := signal.NotifyContext(b.Context(), os.Interrupt)
						defer cancel()
						assert.NoError(b, col.Collector.Run(ctx))
					}()

					require.Eventually(b, func() bool {
						return col.Collector.GetState() == otelcol.StateRunning
					}, 10*time.Second, 1*time.Millisecond, "Collector did not start in time")

					assert.EventuallyWithT(b,
						func(ct *assert.CollectT) {
							debugExporterLogs := col.
								ObservedLogs().
								FilterMessageSnippet(`"message":"event-from-benchmark-input"`).
								All()
							benchmarkEventLogsCount := 0
							for _, logRecord := range debugExporterLogs {
								benchmarkEventLogsCount += strings.Count(logRecord.Message, `"message":"event-from-benchmark-input"`)
							}
							assert.Equal(ct, numEvents, benchmarkEventLogsCount, "expected exactly %d benchmark event logs with custom_field", numEvents)
						},
						10*time.Second,
						2*time.Millisecond)

					b.StopTimer()
					col.Collector.Shutdown()
					wg.Wait()

					if b.Failed() {
						b.Log("OTel Collector logs:\n" + col.AllLogs.String())
					}
					b.StartTimer()
				}
			})
		}
	}
}
