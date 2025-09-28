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

package beatconverter

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap"
	"gopkg.in/yaml.v2"
)

var esCommonOutput = `
extensions:
  beatsauth:
    idle_connection_timeout: 3s
    proxy_disable: false
    timeout: 1m30s
exporters:
  elasticsearch:
    endpoints:
      - https://localhost:9200
    logs_index: form-otel-exporter
    password: changeme
    retry:
      enabled: true
      initial_interval: 1s
      max_interval: 1m0s
      max_retries: 3
    user: elastic
    max_conns_per_host: 1
    sending_queue:
      batch:
        flush_timeout: 10s
        max_size: 1600
        min_size: 0
        sizer: items
      block_on_overflow: true
      enabled: true
      num_consumers: 1
      queue_size: 3200
      wait_for_result: true
    mapping:
      mode: bodymap
    compression: gzip
    compression_params:
      level: 1 
    auth:
      authenticator: beatsauth  
`

func TestConverter(t *testing.T) {
	c := Converter{}
	t.Run("test converter functionality", func(t *testing.T) {
		var supportedInput = `
receivers:
  filebeatreceiver:
    filebeat:
      inputs:
        - type: filestream
          enabled: true
          id: filestream-input-id
          paths:
            - /tmp/flog.log
        - type: log
          enabled: true
          paths:
            - /var/log/*.log
    output:
      elasticsearch:
        hosts: ["https://localhost:9200"]
        username: elastic
        password: changeme
        index: form-otel-exporter

service:
  pipelines:
    logs:
      receivers:
        - "filebeatreceiver"
`
		var expectedOutput = esCommonOutput + `
receivers:
  filebeatreceiver:
    filebeat:
      inputs:
        - enabled: true
          id: filestream-input-id
          paths:
            - /tmp/flog.log
          type: filestream
        - type: log
          enabled: true
          paths:
            - /var/log/*.log
    output:
      otelconsumer: null
service:
  extensions:
    - beatsauth
  pipelines:
    logs:
      exporters:
        - elasticsearch
      receivers:
        - filebeatreceiver
`

		input := newFromYamlString(t, supportedInput)
		err := c.Convert(context.Background(), input)
		require.NoError(t, err, "error converting beats output config")

		expOutput := newFromYamlString(t, expectedOutput)
		compareAndAssert(t, expOutput, input)

	})

	t.Run("test failure if unsupported config is provided", func(t *testing.T) {
		var unsupportedOutputConfig = `
receivers:
  filebeatreceiver:
    filebeat:
      inputs:
        - type: filestream
          enabled: true
          id: filestream-input-id
          paths:
            - /tmp/flog.log
    output:
      kafka:
        enabled: true

service:
  pipelines:
    logs:
      receivers:
        - "filebeatreceiver"
`

		input := newFromYamlString(t, unsupportedOutputConfig)
		err := c.Convert(context.Background(), input)
		require.ErrorContains(t, err, "output type \"kafka\" is unsupported in OTel mode")

	})

	t.Run("test cloud id conversion", func(t *testing.T) {
		var supportedInput = `
receivers:
  filebeatreceiver:
    filebeat:
      inputs:
        - type: filestream
          enabled: true
          id: filestream-input-id
          paths:
            - /tmp/flog.log
    output:
      elasticsearch:
        hosts: ["https://localhost:9200"]
        username: elastic
        password: changeme
        index: form-otel-exporter
    cloud:
      id: ZWxhc3RpYy5jbyRlcy1ob3N0bmFtZSRraWJhbmEtaG9zdG5hbWU=
      auth: elastic-cloud:password
service:
  pipelines:
    logs:
      receivers:
        - "filebeatreceiver"
`
		var expectedOutput = `
extensions:
  beatsauth:
    idle_connection_timeout: 3s
    proxy_disable: false
    timeout: 1m30s
exporters:
  elasticsearch:
    endpoints:
      - https://es-hostname.elastic.co:443
    logs_index: form-otel-exporter
    password: password
    retry:
      enabled: true
      initial_interval: 1s
      max_interval: 1m0s
      max_retries: 3
    user: elastic-cloud
    max_conns_per_host: 1
    sending_queue:
      batch:
        flush_timeout: 10s
        max_size: 1600
        min_size: 0
        sizer: items
      block_on_overflow: true
      enabled: true
      num_consumers: 1
      queue_size: 3200
      wait_for_result: true
    mapping:
      mode: bodymap
    compression: gzip
    compression_params:
      level: 1
    auth:
      authenticator: beatsauth  
receivers:
  filebeatreceiver:
    filebeat:
      inputs:
        - enabled: true
          id: filestream-input-id
          paths:
            - /tmp/flog.log
          type: filestream
    output:
      otelconsumer: null
    cloud: null
    setup:
      kibana:
        host: https://kibana-hostname.elastic.co:443
service:
  extensions:
    - beatsauth
  pipelines:
    logs:
      exporters:
        - elasticsearch
      receivers:
        - filebeatreceiver
`

		input := newFromYamlString(t, supportedInput)
		err := c.Convert(context.Background(), input)
		require.NoError(t, err, "error converting beats output config")

		expOutput := newFromYamlString(t, expectedOutput)
		compareAndAssert(t, expOutput, input)

	})

	t.Run("test local queue setting is promoted to global level", func(t *testing.T) {
		var supportedInput = `
receivers:
  filebeatreceiver:
    output:
      elasticsearch:
        hosts: ["https://localhost:9200"]
        username: elastic
        password: changeme
        index: form-otel-exporter
        queue:
          mem:
            events: 3200
            flush:
              min_events: 1600
              timeout: 10s

service:
  pipelines:
    logs:
      receivers:
        - "filebeatreceiver"
`

		var expectedOutput = esCommonOutput + `
receivers:
  filebeatreceiver:
    queue:
      mem:
        events: 3200
        flush:
          min_events: 1600
          timeout: 10s
    output:
      otelconsumer: null
service:
  extensions:
    - beatsauth
  pipelines:
    logs:
      exporters:
        - elasticsearch
      receivers:
        - filebeatreceiver
`

		input := newFromYamlString(t, supportedInput)
		err := c.Convert(context.Background(), input)
		require.NoError(t, err, "error converting beats output config")

		expOutput := newFromYamlString(t, expectedOutput)
		compareAndAssert(t, expOutput, input)

	})

	t.Run("test logstash exporter", func(t *testing.T) {
		var supportedInput = `
receivers:
  filebeatreceiver:
    output:
      logstash:
        bulk_max_size: 1024
        backoff:
          init: 2s
          max: 2m0s
        compression_level: 9
        escape_html: true
        hosts: ["https://localhost:5044"]
        index: "filebeat"
        loadbalance: true
        max_retries: 2
        pipelining: 0
        proxy_url: "socks5://user:password@socks5-proxy:2233"
        proxy_use_local_resolver: true
        slow_start: true
        # timeout: 30s
        # ttl: 10s
        workers: 2
service:
  pipelines:
    logs:
      receivers:
        - "filebeatreceiver"
`

		var expectedOutput = `
exporters:
  logstash:
    bulk_max_size: 1024
    backoff:
      init: 2s
      max: 2m0s
    compression_level: 9
    escape_html: true
    hosts: ["https://localhost:5044"]
    index: "filebeat"
    loadbalance: true
    max_retries: 2
    pipelining: 0
    proxy_url: "socks5://user:password@socks5-proxy:2233"
    proxy_use_local_resolver: true
    slow_start: true
    timeout: 30s
    ttl: 0s
    worker: 0
    workers: 2
receivers:
  filebeatreceiver:
    output:
      otelconsumer: null
service:
  pipelines:
    logs:
      exporters:
        - logstash
      receivers:
        - filebeatreceiver
`
		input := newFromYamlString(t, supportedInput)
		err := c.Convert(context.Background(), input)
		require.NoError(t, err, "error converting beats logstash-output config")

		expOutput := newFromYamlString(t, expectedOutput)
		compareAndAssert(t, expOutput, input)
	})

	t.Run("logstash config tests queue setting is promoted to global level", func(t *testing.T) {
		var supportedInput = `
receivers:
  filebeatreceiver:
    output:
      logstash:
        hosts: ["https://localhost:5044"]
        queue:
          mem:
            events: 3200
            flush:
              min_events: 1600
              timeout: 10s
service:
  pipelines:
    logs:
      receivers:
        - "filebeatreceiver"
`

		var expectedOutput = `
exporters:
  logstash:
    bulk_max_size: 2048
    backoff:
      init: 1s
      max: 1m0s
    compression_level: 3
    escape_html: false
    hosts: ["https://localhost:5044"]
    index: ""
    loadbalance: false
    max_retries: 3
    pipelining: 2
    proxy_url: ""
    proxy_use_local_resolver: false
    slow_start: false
    timeout: 30s
    ttl: 0s
    worker: 0
    workers: 0
receivers:
  filebeatreceiver:
    queue:
      mem:
        events: 3200
        flush:
          min_events: 1600
          timeout: 10s
    output:
      otelconsumer: null
service:
  pipelines:
    logs:
      exporters:
        - logstash
      receivers:
        - filebeatreceiver
`
		input := newFromYamlString(t, supportedInput)
		err := c.Convert(context.Background(), input)
		require.NoError(t, err, "error converting beats logstash-output config")

		expOutput := newFromYamlString(t, expectedOutput)
		compareAndAssert(t, expOutput, input)
	})

	t.Run("test logstash exporter with enabled false", func(t *testing.T) {
		var supportedInput = `
receivers:
  filebeatreceiver:
    output:
      logstash:
        enabled: false
        hosts: ["https://localhost:5044"]
service:
  pipelines:
    logs:
      receivers:
        - "filebeatreceiver"
`

		var expectedOutput = `
receivers:
  filebeatreceiver:
    output:
      otelconsumer: null
service:
  pipelines:
    logs:
      receivers:
        - filebeatreceiver
`
		input := newFromYamlString(t, supportedInput)
		err := c.Convert(context.Background(), input)
		require.NoError(t, err, "error converting beats logstash-output config")

		expOutput := newFromYamlString(t, expectedOutput)
		compareAndAssert(t, expOutput, input)
	})

	t.Run("test Logstash failure if host is empty", func(t *testing.T) {
		var unsupportedOutputConfig = `
receivers:
  filebeatreceiver:
    output:
      logstash:
service:
  pipelines:
    logs:
      receivers:
        - "filebeatreceiver"
`

		input := newFromYamlString(t, unsupportedOutputConfig)
		err := c.Convert(context.Background(), input)
		require.ErrorContains(t, err, "failed unpacking logstash config: missing required field accessing 'hosts'")

	})
}

func TestLogLevel(t *testing.T) {
	c := Converter{}
	tests := []struct {
		name          string
		level         string
		expectedLevel string
		expectedError string
	}{
		{
			name:          "test-debug",
			level:         "debug",
			expectedLevel: "DEBUG",
		},
		{
			name:          "test-info",
			level:         "info",
			expectedLevel: "INFO",
		},
		{
			name:          "test-warn",
			level:         "warning",
			expectedLevel: "WARN",
		},
		{
			name:          "test-error",
			level:         "error",
			expectedLevel: "ERROR",
		},
		{
			name:          "test-critical",
			level:         "critical",
			expectedLevel: "ERROR",
		},
		{
			name:          "test-error",
			level:         "blabla",
			expectedError: "unrecognized level: blabla",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			supportedInput := fmt.Sprintf(`
      receivers:
        filebeatreceiver:
          logging:
            level: %s
          filebeat:
            inputs:
              - type: filestream
                enabled: true
                id: filestream-input-id
                paths:
                  - /tmp/flog.log
      `, test.level)
			input := newFromYamlString(t, supportedInput)
			err := c.Convert(context.Background(), input)
			if test.expectedError != "" {
				require.ErrorContains(t, err, test.expectedError)
			} else {
				require.NoError(t, err)
				inputMap := input.Get("service::telemetry::logs::level")
				require.Equal(t, test.expectedLevel, inputMap)
			}
		})
	}

}

// when presets are configured, `ToOTelConfig` may override certain fields based on preset values
// such as worker, bulk_max_size, idle_connection_timeout, queue settings etc
// This test ensures correct values of idle_connection_timeout, which is an http config, is configured on beastauth extension
// Also tests correct queue config is set under filebeatreceiver
func TestPresets(t *testing.T) {
	c := Converter{}

	commonBeatCfg := `
receivers:
  filebeatreceiver:
    output:
      elasticsearch:
        hosts:
          - localhost:9200
        index: "some-index"
        username: elastic
        password: changeme
        preset: %s
service:
  pipelines:
    logs:
      receivers:
        - filebeatreceiver  
`

	commonOTelCfg := `
extensions:
  beatsauth:
    idle_connection_timeout: 3s
    proxy_disable: false
    timeout: 1m30s  
receivers:
  filebeatreceiver:
    output:
      otelconsumer: null    
exporters:
  elasticsearch:
    endpoints:
      - http://localhost:9200
    retry:
      enabled: true
      initial_interval: 1s
      max_interval: 1m0s
      max_retries: 3
    logs_index: some-index
    password: changeme
    user: elastic
    mapping:
      mode: bodymap 
    compression: gzip
    compression_params:
      level: 1
    auth:
      authenticator: beatsauth
    sending_queue:
      batch:
        max_size: 1600
        min_size: 0
        sizer: items
      block_on_overflow: true
      enabled: true
      num_consumers: 1
      queue_size: 3200
      wait_for_result: true
service:
  extensions:
    - beatsauth
  pipelines:
    logs:
      exporters:
        - elasticsearch
      receivers:
        - filebeatreceiver      
`

	tests := []struct {
		presetName string
		output     string
	}{
		{
			presetName: "balanced",
			output: commonOTelCfg + `
receivers:
  filebeatreceiver:
    queue:
      mem:
        events: 3200
        flush:
          min_events: 1600
          timeout: 10s            
extensions:
  beatsauth:
    idle_connection_timeout: 3s
exporters:
  elasticsearch:
    sending_queue:
      batch:
        flush_timeout: 10s
        max_size: 1600
      num_consumers: 1
      queue_size: 3200
    max_conns_per_host: 1      
 `,
		},
		{
			presetName: "throughput",
			output: commonOTelCfg + `
receivers:
  filebeatreceiver:
    queue:
      mem:
        events: 12800
        flush:
          min_events: 1600
          timeout: 5s           
extensions:
  beatsauth:
    idle_connection_timeout: 15s
exporters:
  elasticsearch:
    sending_queue:
      batch:
        flush_timeout: 10s
        max_size: 1600
      num_consumers: 4
      queue_size: 12800
    max_conns_per_host: 4      
`,
		},
		{
			presetName: "scale",
			output: `
receivers:
  filebeatreceiver:
    queue:
      mem:
        events: 3200
        flush:
          min_events: 1600
          timeout: 20s          
extensions:
  beatsauth:
    idle_connection_timeout: 1s
exporters:
  elasticsearch:
    sending_queue:
      batch:
        flush_timeout: 10s
        max_size: 1600
      num_consumers: 1
      queue_size: 3200
    max_conns_per_host: 1
    retry:
      initial_interval: 5s
      max_interval: 5m0s       
`,
		},
		{
			presetName: "latency",
			output: commonOTelCfg + `
receivers:
  filebeatreceiver:
    queue:
      mem:
        events: 4100
        flush:
          min_events: 2050
          timeout: 1s          
extensions:
  beatsauth:
    idle_connection_timeout: 1m0s
exporters:
  elasticsearch:
    sending_queue:
      batch:
        flush_timeout: 10s
        max_size: 50
      num_consumers: 1
      queue_size: 4100
    max_conns_per_host: 1
    retry:
      initial_interval: 1s
      max_interval: 1m0s    
`}}

	commonOTeMap := newFromYamlString(t, commonOTelCfg)

	for _, test := range tests {
		t.Run("config translation w/"+test.presetName, func(t *testing.T) {
			cfg := newFromYamlString(t, fmt.Sprintf(commonBeatCfg, test.presetName))
			err := c.Convert(t.Context(), cfg)
			require.NoError(t, err, "error converting beats output config")
			expOutput := newFromYamlString(t, test.output)
			err = commonOTeMap.Merge(expOutput)
			require.NoError(t, err)
			compareAndAssert(t, commonOTeMap, cfg)
		})
	}

}
func newFromYamlString(t *testing.T, input string) *confmap.Conf {
	t.Helper()
	var rawConf map[string]any
	err := yaml.Unmarshal([]byte(input), &rawConf)
	require.NoError(t, err)

	return confmap.NewFromStringMap(rawConf)
}

func compareAndAssert(t *testing.T, expectedOutput *confmap.Conf, gotOutput *confmap.Conf) {
	t.Helper()
	// convert it to a common type
	want, err := yaml.Marshal(expectedOutput.ToStringMap())
	require.NoError(t, err)
	got, err := yaml.Marshal(gotOutput.ToStringMap())
	require.NoError(t, err)

	assert.Equal(t, string(want), string(got))
}
