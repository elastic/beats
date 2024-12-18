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

package fbprovider

import (
	"context"
	_ "embed"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap"

	"gopkg.in/yaml.v2"
)

var beatsConfig = `
filebeat.inputs:
  - type: filestream
    id: filestream-input-id
    enabled: true
    paths:
      - /tmp/flog.log

output:
  elasticsearch:
    hosts: ["https://localhost:9200"]
    username: elastic
    password: changeme
    index: form-otel-exporter
    ssl.enabled: false

setup.template.name: form-otel-exporter
setup.template.pattern: form-otel-exporter
setup.dashboards.index: "form-otel-exporter*"

setup.kibana:
  host: https://localhost:5601
  username: elastic
  password: changeme
  ssl.verification_mode: none
`

var expectedOutput = `
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
      elasticsearch:
        hosts: ["https://localhost:9200"]
        username: elastic
        password: changeme
        index: form-otel-exporter
        ssl:
          enabled: false
    path:
      config: .
      data: ./data
      home: .
      logs: ./logs
    setup:
      dashboards:
        index: form-otel-exporter*
      kibana:
        host: https://localhost:5601
        password: changeme
        ssl:
          verification_mode: none
        username: elastic
      template:
        name: form-otel-exporter
        pattern: form-otel-exporter

service:
  pipelines:
    logs:
      receivers:
        - "filebeatreceiver"
`

func TestFileBeatProvider(t *testing.T) {
	p := provider{}

	t.Run("test filebeat provider", func(t *testing.T) {

		tempFile, err := os.CreateTemp("", "filebeat.yml")
		require.NoError(t, err, "error creating temp file")
		defer os.Remove(tempFile.Name()) // Clean up the file after we're done
		defer tempFile.Close()

		content := []byte(beatsConfig)
		_, err = tempFile.Write(content)
		require.NoError(t, err, "error creating temp file")

		// prefix file path with fb:
		ret, err := p.Retrieve(context.Background(), "fb:"+tempFile.Name(), nil)
		require.NoError(t, err)

		retValue, err := ret.AsRaw()
		require.NoError(t, err)
		expOutput := newFromYamlString(t, expectedOutput)

		// convert it into a common type
		want, err := yaml.Marshal(expOutput.ToStringMap())
		require.NoError(t, err)
		got, err := yaml.Marshal(retValue)
		require.NoError(t, err)

		assert.Equal(t, string(want), string(got))
		assert.NoError(t, p.Shutdown(context.Background()))
	})

}

func newFromYamlString(t *testing.T, input string) *confmap.Conf {
	t.Helper()
	var rawConf map[string]any
	err := yaml.Unmarshal([]byte(input), &rawConf)
	require.NoError(t, err)

	return confmap.NewFromStringMap(rawConf)
}
